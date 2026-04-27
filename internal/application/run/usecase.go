package run

import (
	"context"
	"fmt"
	"strings"
	"time"

	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"
	"github.com/EdOoO21/metadata-parser/internal/ports"
	"github.com/EdOoO21/metadata-parser/internal/settings"
)

type ExecuteInput struct {
	Repository         appports.CatalogRepository
	Config             *settings.AppConfig
	ConfigHash         string
	ConfigSnapshotJSON []byte
}

type RunCatalogUseCase struct {
	logger          ports.Logger
	sourceProcessor *SourceProcessor
}

type CompletedWithErrorsError struct {
	RunID  int64
	Errors []string
}

func (e *CompletedWithErrorsError) Error() string {
	return fmt.Sprintf("run completed with errors: %s", strings.Join(e.Errors, "; "))
}

func NewRunCatalogUseCase(
	logger ports.Logger,
	sourceProcessor *SourceProcessor,
) *RunCatalogUseCase {
	return &RunCatalogUseCase{
		logger:          logger,
		sourceProcessor: sourceProcessor,
	}
}

func (uc *RunCatalogUseCase) Execute(ctx context.Context, input ExecuteInput) (int64, error) {
	cfg := input.Config

	uc.logger.Info("configuration loaded",
		"version", cfg.Version,
		"catalog_dsn_env", cfg.Catalog.DSNEnv,
		"source_count", len(cfg.Sources),
	)

	repo := input.Repository

	run, err := repo.CreateRun(ctx, model.Run{
		StartedAt:          time.Now().UTC(),
		Status:             types.RunStatusRunning,
		ConfigHash:         input.ConfigHash,
		ConfigSnapshotJSON: input.ConfigSnapshotJSON,
	})
	if err != nil {
		return 0, fmt.Errorf("create run: %w", err)
	}

	uc.logger.Info("catalog run created",
		"run_id", run.ID,
		"status", string(run.Status),
	)

	var processingErrors []string
	var successCount int
	var failureCount int

	for _, src := range cfg.Sources {
		if err := uc.sourceProcessor.Process(ctx, repo, run.ID, src); err != nil {
			failureCount++
			processingErrors = append(processingErrors, fmt.Sprintf("%s: %v", src.Name, err))

			uc.logger.Error("source processing failed",
				"name", src.Name,
				"kind", src.Kind,
				"error", err.Error(),
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
		return 0, fmt.Errorf("update run status: %w", err)
	}

	uc.logger.Info("catalog run finished",
		"run_id", run.ID,
		"status", string(runStatus),
		"successful_sources", successCount,
		"failed_sources", failureCount,
	)

	if len(processingErrors) > 0 {
		return run.ID, &CompletedWithErrorsError{
			RunID:  run.ID,
			Errors: append([]string(nil), processingErrors...),
		}
	}

	return run.ID, nil
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
