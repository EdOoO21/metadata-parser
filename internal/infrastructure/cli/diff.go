package cli

import (
	"context"
	"fmt"

	diffapp "github.com/EdOoO21/metadata-parser/internal/application/diff"
	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/spf13/cobra"
)

func NewDiffCmd(
	configLoader ConfigLoader,
	catalogOpener appports.CatalogRepositoryOpener,
	diffCatalogUseCase DiffCatalogUseCase,
) *cobra.Command {
	var latest bool
	var fromRunID int64
	var toRunID int64
	var configPath string

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Сравнить два слепка каталога и показать различия",
		Long: `Команда читает каталог по двум выбранным запускам
и показывает добавленные, удаленные и измененные датасеты и колонки.`,
		Example: `  catalog diff --config ./demo/config/demo.yaml --from-run-id 41 --to-run-id 42`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if configLoader == nil {
				return fmt.Errorf("config loader is not configured")
			}
			if catalogOpener == nil {
				return fmt.Errorf("catalog opener is not configured")
			}
			if diffCatalogUseCase == nil {
				return fmt.Errorf("diff catalog use case is not configured")
			}
			if err := validateDiffSelection(latest, fromRunID, toRunID); err != nil {
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
			resolvedFromRunID, resolvedToRunID, err := resolveDiffRunIDs(cmd.Context(), repo, latest, fromRunID, toRunID)
			if err != nil {
				return err
			}

			message, err := diffCatalogUseCase.Execute(cmd.Context(), diffapp.ExecuteInput{
				Repository: repo,
				FromRunID:  resolvedFromRunID,
				ToRunID:    resolvedToRunID,
			})
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), message)
			return nil
		},
	}

	cmd.Flags().BoolVar(&latest, "latest", false, "Сравнить два последних запуска")
	cmd.Flags().Int64Var(&fromRunID, "from-run-id", 0, "Идентификатор исходного запуска")
	cmd.Flags().Int64Var(&toRunID, "to-run-id", 0, "Идентификатор целевого запуска")
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Путь к YAML-конфигу")
	_ = cmd.MarkFlagRequired("config")

	return cmd
}

func validateDiffSelection(latest bool, fromRunID, toRunID int64) error {
	if latest && (fromRunID > 0 || toRunID > 0) {
		return fmt.Errorf("flags --latest and --from-run-id/--to-run-id cannot be used together")
	}
	if !latest && (fromRunID <= 0 || toRunID <= 0) {
		return fmt.Errorf("flags --from-run-id and --to-run-id are required")
	}
	return nil
}

type diffRecentRunsLister interface {
	ListRecentRuns(ctx context.Context, limit int) ([]model.Run, error)
}

func resolveDiffRunIDs(
	ctx context.Context,
	repo appports.CatalogRepository,
	latest bool,
	fromRunID int64,
	toRunID int64,
) (int64, int64, error) {
	if !latest {
		return fromRunID, toRunID, nil
	}

	lister, ok := repo.(diffRecentRunsLister)
	if !ok {
		return 0, 0, fmt.Errorf("latest run selector is not supported by the catalog repository")
	}

	runs, err := lister.ListRecentRuns(ctx, 2)
	if err != nil {
		return 0, 0, err
	}
	if len(runs) < 2 {
		return 0, 0, fmt.Errorf("at least two runs are required for --latest")
	}

	return runs[1].ID, runs[0].ID, nil
}
