package cli

import (
	"fmt"

	diffapp "github.com/EdOoO21/metadata-parser/internal/application/diff"
	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
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

			message, err := diffCatalogUseCase.Execute(cmd.Context(), diffapp.ExecuteInput{
				Repository: catalogConn.Repository(),
				FromRunID:  fromRunID,
				ToRunID:    toRunID,
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
	if latest {
		panic("diff latest selector is not implemented yet")
	}
	if fromRunID <= 0 || toRunID <= 0 {
		return fmt.Errorf("flags --from-run-id and --to-run-id are required")
	}
	return nil
}
