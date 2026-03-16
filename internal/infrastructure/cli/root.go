package cli

import "github.com/spf13/cobra"

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "Catalog heterogeneous data sources into a unified metadata store",
	}

	cmd.AddCommand(NewRunCmd())
	cmd.AddCommand(NewReportCmd())
	cmd.AddCommand(NewDiffCmd())

	return cmd
}
