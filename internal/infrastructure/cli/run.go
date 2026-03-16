package cli

import (
	"fmt"
	"log/slog"

	appconfig "catalog-tool/internal/infrastructure/config"
	"catalog-tool/internal/infrastructure/logging"
	"github.com/spf13/cobra"
)

func NewRunCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Collect metadata from configured sources and persist a new snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := logging.NewLogger()
			loader := appconfig.NewLoader()

			cfg, err := loader.Load(configPath)
			if err != nil {
				return err
			}

			logger.Info("loaded config",
				slog.Int("version", cfg.Version),
				slog.String("catalog_dsn_env", cfg.Catalog.DSNEnv),
				slog.Int("source_count", len(cfg.Sources)),
			)

			for _, src := range cfg.Sources {
				logger.Info("configured source",
					slog.String("name", src.Name),
					slog.String("kind", src.Kind),
				)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "run bootstrap completed")
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to YAML config")
	_ = cmd.MarkFlagRequired("config")

	return cmd
}
