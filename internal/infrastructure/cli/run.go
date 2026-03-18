package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/EdOoO21/metadata-parser/internal/application/dto"
	"github.com/EdOoO21/metadata-parser/internal/application/ports"
	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"
	appconfig "github.com/EdOoO21/metadata-parser/internal/infrastructure/config"
	filescsv "github.com/EdOoO21/metadata-parser/internal/infrastructure/connectors/filescsv"
	"github.com/EdOoO21/metadata-parser/internal/infrastructure/db/postgres"
	"github.com/EdOoO21/metadata-parser/internal/infrastructure/logging"

	"github.com/spf13/cobra"
)

const filePreviewMaxRows = 10

func NewRunCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Прочитать YAML-конфиг, пройтись по источникам и сохранить слепок в каталог",
		Long: `Команда загружает YAML-конфиг, валидирует его,
обходит поддержанные источники данных и сохраняет слепок в PostgreSQL-каталог.`,
		Example: `  catalog run --config ./demo/config/demo.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			logger := logging.NewLogger()
			loader := appconfig.NewLoader()

			cfg, err := loader.Load(configPath)
			if err != nil {
				return err
			}

			configSnapshotJSON, err := json.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("marshal config snapshot: %w", err)
			}

			logger.Info("configuration loaded",
				slog.Int("version", cfg.Version),
				slog.String("catalog_dsn_env", cfg.Catalog.DSNEnv),
				slog.Int("source_count", len(cfg.Sources)),
			)

			pool, err := postgres.NewPoolFromEnv(ctx, cfg.Catalog.DSNEnv)
			if err != nil {
				return err
			}
			defer pool.Close()

			repo := postgres.NewRepository(pool)

			run, err := repo.CreateRun(ctx, model.Run{
				StartedAt:          time.Now().UTC(),
				Status:             types.RunStatusRunning,
				ConfigHash:         hashBytes(configSnapshotJSON),
				ConfigSnapshotJSON: configSnapshotJSON,
			})
			if err != nil {
				return fmt.Errorf("create run: %w", err)
			}

			logger.Info("catalog run created",
				slog.Int64("run_id", run.ID),
				slog.String("status", string(run.Status)),
			)

			previewer := filescsv.NewPreviewCSVService(
				filescsv.NewFileSourceResolver(
					filescsv.NewLocalFileSource(),
				),
				filescsv.NewCSVConnector(),
			)

			var processingErrors []string
			var successCount int
			var failureCount int

			for _, src := range cfg.Sources {
				if err := processSource(ctx, logger, repo, previewer, run.ID, src); err != nil {
					failureCount++
					processingErrors = append(processingErrors, fmt.Sprintf("%s: %v", src.Name, err))

					logger.Error("source processing failed",
						slog.String("name", src.Name),
						slog.String("kind", src.Kind),
						slog.String("error", err.Error()),
					)

					continue
				}

				successCount++
			}

			runFinishedAt := time.Now().UTC()
			runStatus := determineRunStatus(successCount, failureCount)

			var runErrorMessage *string
			if len(processingErrors) > 0 {
				msg := strings.Join(processingErrors, "; ")
				runErrorMessage = &msg
			}

			if err := repo.UpdateRunStatus(ctx, run.ID, runStatus, &runFinishedAt, runErrorMessage); err != nil {
				return fmt.Errorf("update run status: %w", err)
			}

			logger.Info("catalog run finished",
				slog.Int64("run_id", run.ID),
				slog.String("status", string(runStatus)),
				slog.Int("successful_sources", successCount),
				slog.Int("failed_sources", failureCount),
			)

			if len(processingErrors) > 0 {
				return fmt.Errorf("run completed with errors: %s", strings.Join(processingErrors, "; "))
			}

			fmt.Fprintf(cmd.OutOrStdout(), "run completed successfully: run_id=%d\n", run.ID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Путь к YAML-конфигу")
	_ = cmd.MarkFlagRequired("config")

	return cmd
}

func processSource(
	ctx context.Context,
	logger *slog.Logger,
	repo ports.CatalogRepository,
	previewer *filescsv.PreviewCSVService,
	runID int64,
	src dto.SourceConfig,
) error {
	logger.Info("configured source discovered",
		slog.String("name", src.Name),
		slog.String("kind", src.Kind),
	)

	source, err := repo.EnsureSource(ctx, model.Source{
		Name: src.Name,
		Kind: types.SourceKind(src.Kind),
	})
	if err != nil {
		return fmt.Errorf("ensure source: %w", err)
	}

	effectiveConfigJSON, err := json.Marshal(src.Config)
	if err != nil {
		return fmt.Errorf("marshal effective config: %w", err)
	}

	runSource, err := repo.CreateRunSource(ctx, model.RunSource{
		RunID:               runID,
		SourceID:            source.ID,
		StartedAt:           time.Now().UTC(),
		Status:              types.RunStatusRunning,
		EffectiveConfigJSON: effectiveConfigJSON,
	})
	if err != nil {
		return fmt.Errorf("create run source: %w", err)
	}

	processErr := handleSourceByKind(ctx, logger, repo, previewer, runSource.ID, src)

	finishedAt := time.Now().UTC()

	if processErr != nil {
		msg := processErr.Error()
		if err := repo.UpdateRunSourceStatus(ctx, runSource.ID, types.RunStatusFailed, &finishedAt, &msg); err != nil {
			return fmt.Errorf("processing error: %v; update run source status: %w", processErr, err)
		}
		return processErr
	}

	if err := repo.UpdateRunSourceStatus(ctx, runSource.ID, types.RunStatusSuccess, &finishedAt, nil); err != nil {
		return fmt.Errorf("update run source status: %w", err)
	}

	logger.Info("source processed successfully",
		slog.String("name", src.Name),
		slog.String("kind", src.Kind),
		slog.Int64("run_source_id", runSource.ID),
	)

	return nil
}

func handleSourceByKind(
	ctx context.Context,
	logger *slog.Logger,
	repo ports.CatalogRepository,
	previewer *filescsv.PreviewCSVService,
	runSourceID int64,
	src dto.SourceConfig,
) error {
	switch src.Kind {
	case string(types.SourceKindFiles):
		return processFileSource(ctx, logger, repo, previewer, runSourceID, src)

	default:
		return fmt.Errorf("source kind %q is not implemented yet", src.Kind)
	}
}

func processFileSource(
	ctx context.Context,
	logger *slog.Logger,
	repo ports.CatalogRepository,
	previewer *filescsv.PreviewCSVService,
	runSourceID int64,
	src dto.SourceConfig,
) error {
	path := strings.TrimSpace(src.Config.Path)
	ext := strings.ToLower(filepath.Ext(path))

	if ext != ".csv" {
		return fmt.Errorf("unsupported file extension %q for files source", ext)
	}

	result, err := previewer.Execute(ctx, filescsv.LocalFile(path), filePreviewMaxRows)
	if err != nil {
		return fmt.Errorf("preview csv: %w", err)
	}

	metadataJSON, err := json.Marshal(map[string]any{
		"format":            "csv",
		"headers":           result.Headers,
		"preview_row_count": len(result.Rows),
		"preview_max_rows":  filePreviewMaxRows,
	})
	if err != nil {
		return fmt.Errorf("marshal dataset metadata: %w", err)
	}

	datasetName := filepath.Base(path)
	if strings.TrimSpace(datasetName) == "" {
		datasetName = src.Name
	}

	dataset, err := repo.CreateDataset(ctx, model.Dataset{
		RunSourceID:   runSourceID,
		Kind:          types.DatasetKindFile,
		DatasetKey:    path,
		Name:          datasetName,
		Location:      path,
		DiscoveredAt:  time.Now().UTC(),
		ProfileStatus: types.ProfileStatusDiscoveredOnly,
		MetadataJSON:  metadataJSON,
	})
	if err != nil {
		return fmt.Errorf("create dataset: %w", err)
	}

	for i, header := range result.Headers {
		_, err := repo.CreateColumn(ctx, model.Column{
			DatasetID:       dataset.ID,
			Name:            header,
			OriginalType:    "csv",
			NormalizedType:  types.CanonicalTypeString,
			IsNullable:      true,
			OrdinalPosition: i + 1,
		})
		if err != nil {
			return fmt.Errorf("create column %q: %w", header, err)
		}
	}

	logger.Info("csv source preview completed",
		slog.String("name", src.Name),
		slog.String("path", path),
		slog.Int64("dataset_id", dataset.ID),
		slog.Int("preview_row_count", len(result.Rows)),
		slog.Int("preview_max_rows", filePreviewMaxRows),
		slog.Int("header_count", len(result.Headers)),
		slog.String("headers", strings.Join(result.Headers, ",")),
	)

	return nil
}

func determineRunStatus(successCount, failureCount int) types.RunStatus {
	switch {
	case failureCount == 0:
		return types.RunStatusSuccess
	case successCount == 0:
		return types.RunStatusFailed
	default:
		return types.RunStatusPartial
	}
}

func hashBytes(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
