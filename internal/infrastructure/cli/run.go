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
		Short: "Прочитать и проверить YAML-конфиг запуска",
		Long: `Команда загружает YAML-конфиг, проверяет обязательные поля
и показывает найденные источники данных.`,
		Example: `  catalog run --config ./demo/config/demo.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := logging.NewLogger()
			loader := appconfig.NewLoader()

			cfg, err := loader.Load(configPath)
			if err != nil {
				return err
			}

			logger.Info("configuration loaded",
				slog.Int("version", cfg.Version),
				slog.String("catalog_dsn_env", cfg.Catalog.DSNEnv),
				slog.Int("source_count", len(cfg.Sources)),
			)

			for _, src := range cfg.Sources {
				logger.Info("configured source discovered",
					slog.String("name", src.Name),
					slog.String("kind", src.Kind),
				)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "configuration loaded and validated successfully")
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Путь к YAML-конфигу")
	_ = cmd.MarkFlagRequired("config")

	return cmd
}
