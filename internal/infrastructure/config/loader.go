package config

import (
	"fmt"
	"os"

	"catalog-tool/internal/application/dto"
	"gopkg.in/yaml.v3"
)

type Loader struct{}

func NewLoader() *Loader {
	return &Loader{}
}

func (l *Loader) Load(path string) (*dto.AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg dto.AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal yaml: %w", err)
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validateConfig(cfg *dto.AppConfig) error {
	if cfg.Version != 1 {
		return fmt.Errorf("unsupported config version: %d", cfg.Version)
	}
	if cfg.Catalog.DSNEnv == "" {
		return fmt.Errorf("catalog.dsn_env is required")
	}
	if len(cfg.Sources) == 0 {
		return fmt.Errorf("at least one source is required")
	}

	seen := make(map[string]struct{}, len(cfg.Sources))
	for i, src := range cfg.Sources {
		if src.Name == "" {
			return fmt.Errorf("sources[%d].name is required", i)
		}
		if _, ok := seen[src.Name]; ok {
			return fmt.Errorf("duplicate source name: %s", src.Name)
		}
		seen[src.Name] = struct{}{}
		if src.Kind == "" {
			return fmt.Errorf("sources[%d].kind is required", i)
		}
		switch src.Kind {
		case "postgres":
			if src.Config.DSNEnv == "" {
				return fmt.Errorf("sources[%d].config.dsn_env is required for postgres", i)
			}
		case "files":
			if src.Config.Path == "" {
				return fmt.Errorf("sources[%d].config.path is required for files", i)
			}
		case "rest":
			if src.Config.BaseURL == "" {
				return fmt.Errorf("sources[%d].config.base_url is required for rest", i)
			}
			if src.Config.Discovery == nil {
				return fmt.Errorf("sources[%d].config.discovery is required for rest", i)
			}
			if src.Config.Discovery.Mode != "openapi" {
				return fmt.Errorf("sources[%d].config.discovery.mode must be openapi", i)
			}
			if src.Config.Discovery.OpenAPIURL == "" {
				return fmt.Errorf("sources[%d].config.discovery.openapi_url is required for rest", i)
			}
		default:
			return fmt.Errorf("unsupported source kind: %s", src.Kind)
		}
	}
	return nil
}
