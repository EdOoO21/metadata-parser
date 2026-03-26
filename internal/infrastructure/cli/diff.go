package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewDiffCmd(diffCatalogUseCase DiffCatalogUseCase) *cobra.Command {
	var latest bool
	var fromRunID int64
	var toRunID int64

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Сравнить два слепка каталога и показать различия",
		Long: `Команда принимает параметры выбора двух запусков
и подготавливает сравнение слепков каталога.`,
		Example: `  catalog diff --latest
  catalog diff --from-run-id 41 --to-run-id 42`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if diffCatalogUseCase == nil {
				return fmt.Errorf("diff catalog use case is not configured")
			}
			if err := validateDiffSelection(latest, fromRunID, toRunID); err != nil {
				return err
			}

			message, err := diffCatalogUseCase.Execute(cmd.Context(), fromRunID, toRunID)
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
