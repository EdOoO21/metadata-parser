package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewReportCmd() *cobra.Command {
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
			fmt.Fprintf(cmd.OutOrStdout(), "report command received: latest=%t run-id=%d\n", latest, runID)
			return nil
		},
	}

	cmd.Flags().BoolVar(&latest, "latest", false, "Использовать последний доступный запуск")
	cmd.Flags().Int64Var(&runID, "run-id", 0, "Идентификатор запуска")

	return cmd
}
