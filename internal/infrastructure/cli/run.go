package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
	runapp "github.com/EdOoO21/metadata-parser/internal/application/run"
	"github.com/spf13/cobra"
)

func NewRunCmd(
	configLoader ConfigLoader,
	catalogOpener appports.CatalogRepositoryOpener,
	runCatalogUseCase RunCatalogUseCase,
) *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Прочитать YAML-конфиг, пройтись по источникам и сохранить слепок в каталог",
		Long: `Команда загружает YAML-конфиг, валидирует его,
обходит поддержанные источники данных и сохраняет слепок в PostgreSQL-каталог.`,
		Example: `  catalog run --config ./demo/config/demo.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if configLoader == nil {
				return fmt.Errorf("config loader is not configured")
			}
			if catalogOpener == nil {
				return fmt.Errorf("catalog opener is not configured")
			}
			if runCatalogUseCase == nil {
				return fmt.Errorf("run catalog use case is not configured")
			}

			cfg, err := configLoader.Load(configPath)
			if err != nil {
				return err
			}

			configSnapshotJSON, err := json.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("marshal config snapshot: %w", err)
			}

			catalogConn, err := catalogOpener.Open(cmd.Context(), cfg.Catalog.DSNEnv)
			if err != nil {
				return err
			}
			defer catalogConn.Close()

			runID, err := runCatalogUseCase.Execute(cmd.Context(), runapp.ExecuteInput{
				Repository:         catalogConn.Repository(),
				Config:             cfg,
				ConfigHash:         hashBytes(configSnapshotJSON),
				ConfigSnapshotJSON: configSnapshotJSON,
			})
			if err != nil {
				var runErr *runapp.CompletedWithErrorsError
				if errors.As(err, &runErr) {
					fmt.Fprintf(cmd.OutOrStdout(), "run completed with errors: run_id=%d\n", runID)
				}
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "run completed successfully: run_id=%d\n", runID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Путь к YAML-конфигу")
	_ = cmd.MarkFlagRequired("config")

	return cmd
}

func hashBytes(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
