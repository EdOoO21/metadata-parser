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
		Short: "Print a human-readable catalog report to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "report bootstrap: latest=%t run-id=%d\n", latest, runID)
			return nil
		},
	}

	cmd.Flags().BoolVar(&latest, "latest", false, "Use the latest completed run")
	cmd.Flags().Int64Var(&runID, "run-id", 0, "Build report for a specific run id")

	return cmd
}
