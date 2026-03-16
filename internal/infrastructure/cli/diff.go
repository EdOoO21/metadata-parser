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
		Short: "Compare two catalog snapshots and print the differences to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "diff bootstrap: latest=%t from-run-id=%d to-run-id=%d\n", latest, fromRunID, toRunID)
			return nil
		},
	}

	cmd.Flags().BoolVar(&latest, "latest", false, "Compare the two latest completed runs")
	cmd.Flags().Int64Var(&fromRunID, "from-run-id", 0, "Base run id")
	cmd.Flags().Int64Var(&toRunID, "to-run-id", 0, "Target run id")

	return cmd
}
