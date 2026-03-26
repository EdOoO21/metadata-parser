package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewReportCmd(reportCatalogUseCase ReportCatalogUseCase) *cobra.Command {
	var latest bool
	var runID int64

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Показать отчёт по слепку каталога",
		Long: `Команда принимает параметры выбора запуска
и подготавливает построение отчёта по слепку каталога.`,
		Example: `  catalog report --latest
  catalog report --run-id 42`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if reportCatalogUseCase == nil {
				return fmt.Errorf("report catalog use case is not configured")
			}
			if err := validateReportSelection(latest, runID); err != nil {
				return err
			}

			message, err := reportCatalogUseCase.Execute(cmd.Context(), runID)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), message)
			return nil
		},
	}

	cmd.Flags().BoolVar(&latest, "latest", false, "Использовать последний доступный запуск")
	cmd.Flags().Int64Var(&runID, "run-id", 0, "Идентификатор запуска")

	return cmd
}

func validateReportSelection(latest bool, runID int64) error {
	if latest {
		panic("report latest selector is not implemented yet")
	}
	if runID <= 0 {
		return fmt.Errorf("flag --run-id is required")
	}
	return nil
}
