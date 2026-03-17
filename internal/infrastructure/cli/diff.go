package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewDiffCmd() *cobra.Command {
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
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"diff command received: latest=%t from-run-id=%d to-run-id=%d\n",
				latest,
				fromRunID,
				toRunID,
			)
			return nil
		},
	}

	cmd.Flags().BoolVar(&latest, "latest", false, "Сравнить два последних запуска")
	cmd.Flags().Int64Var(&fromRunID, "from-run-id", 0, "Идентификатор исходного запуска")
	cmd.Flags().Int64Var(&toRunID, "to-run-id", 0, "Идентификатор целевого запуска")

	return cmd
}
