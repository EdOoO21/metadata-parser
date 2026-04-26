package cli

import (
	"context"
	"fmt"
	"os"

	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
	reportapp "github.com/EdOoO21/metadata-parser/internal/application/report"
	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/spf13/cobra"
)

func NewReportCmd(
	configLoader ConfigLoader,
	catalogOpener appports.CatalogRepositoryOpener,
	reportCatalogUseCase ReportCatalogUseCase,
) *cobra.Command {
	var latest bool
	var runID int64
	var configPath string
	var outputPath string
	var htmlOutputPath string
	var csvOutputPath string

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Показать отчёт по слепку каталога",
		Long: `Команда читает каталог по выбранному запуску
и строит карту датасетов в Markdown/HTML, а также CSV-экспорт колонок.`,
		Example: `  catalog report --config ./demo/config/demo.yaml --run-id 42
  catalog report --config ./demo/config/demo.yaml --run-id 42 --output ./report.md
  catalog report --config ./demo/config/demo.yaml --run-id 42 --html-output ./report.html
  catalog report --config ./demo/config/demo.yaml --run-id 42 --csv-output ./report.csv`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if configLoader == nil {
				return fmt.Errorf("config loader is not configured")
			}
			if catalogOpener == nil {
				return fmt.Errorf("catalog opener is not configured")
			}
			if reportCatalogUseCase == nil {
				return fmt.Errorf("report catalog use case is not configured")
			}
			if err := validateReportSelection(latest, runID); err != nil {
				return err
			}

			cfg, err := configLoader.Load(configPath)
			if err != nil {
				return err
			}

			catalogConn, err := catalogOpener.Open(cmd.Context(), cfg.Catalog.DSNEnv)
			if err != nil {
				return err
			}
			defer catalogConn.Close()

			repo := catalogConn.Repository()
			resolvedRunID, err := resolveReportRunID(cmd.Context(), repo, latest, runID)
			if err != nil {
				return err
			}

			result, err := reportCatalogUseCase.Execute(cmd.Context(), reportapp.ExecuteInput{
				Repository: repo,
				RunID:      resolvedRunID,
			})
			if err != nil {
				return err
			}

			if outputPath != "" {
				if err := writeFile(outputPath, []byte(result.Markdown)); err != nil {
					return err
				}
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), result.Markdown)
			}

			if htmlOutputPath != "" {
				if err := writeFile(htmlOutputPath, []byte(result.HTML)); err != nil {
					return err
				}
			}

			if csvOutputPath != "" {
				if err := writeFile(csvOutputPath, result.CSV); err != nil {
					return err
				}
			}

			if outputPath != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "markdown report written: %s\n", outputPath)
			}
			if htmlOutputPath != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "html report written: %s\n", htmlOutputPath)
			}
			if csvOutputPath != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "csv export written: %s\n", csvOutputPath)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&latest, "latest", false, "Использовать последний доступный запуск")
	cmd.Flags().Int64Var(&runID, "run-id", 0, "Идентификатор запуска")
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Путь к YAML-конфигу")
	cmd.Flags().StringVar(&outputPath, "output", "", "Путь для сохранения Markdown-отчёта")
	cmd.Flags().StringVar(&htmlOutputPath, "html-output", "", "Путь для сохранения HTML-отчёта")
	cmd.Flags().StringVar(&csvOutputPath, "csv-output", "", "Путь для сохранения CSV-экспорта")
	_ = cmd.MarkFlagRequired("config")

	return cmd
}

func validateReportSelection(latest bool, runID int64) error {
	if latest && runID > 0 {
		return fmt.Errorf("flags --latest and --run-id cannot be used together")
	}
	if !latest && runID <= 0 {
		return fmt.Errorf("flag --run-id is required")
	}
	return nil
}

type recentRunsLister interface {
	ListRecentRuns(ctx context.Context, limit int) ([]model.Run, error)
}

func resolveReportRunID(ctx context.Context, repo appports.CatalogRepository, latest bool, runID int64) (int64, error) {
	if !latest {
		return runID, nil
	}

	lister, ok := repo.(recentRunsLister)
	if !ok {
		return 0, fmt.Errorf("latest run selector is not supported by the catalog repository")
	}

	runs, err := lister.ListRecentRuns(ctx, 1)
	if err != nil {
		return 0, err
	}
	if len(runs) == 0 {
		return 0, fmt.Errorf("no runs found in catalog")
	}

	return runs[0].ID, nil
}

func writeFile(path string, payload []byte) error {
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write file %q: %w", path, err)
	}
	return nil
}
