package settings

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Loader struct{}

const defaultCatalogDSNEnv = "CATALOG_DSN"

func NewLoader() *Loader {
	return &Loader{}
}

func (l *Loader) Load(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal yaml: %w", err)
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validateConfig(cfg *AppConfig) error {
	if cfg.Version != 1 {
		return fmt.Errorf("неподдерживаемая версия конфига: %d", cfg.Version)
	}
	if cfg.Catalog.DSNEnv == "" {
		cfg.Catalog.DSNEnv = defaultCatalogDSNEnv
	}
	if len(cfg.Sources) == 0 {
		return fmt.Errorf("нужно указать хотя бы один источник")
	}

	seen := make(map[string]struct{}, len(cfg.Sources))
	for i, src := range cfg.Sources {
		if src.Name == "" {
			return fmt.Errorf("поле sources[%d].name обязательно", i)
		}
		if _, ok := seen[src.Name]; ok {
			return fmt.Errorf("дублирующееся имя источника: %s", src.Name)
		}
		seen[src.Name] = struct{}{}
		if src.Kind == "" {
			return fmt.Errorf("поле sources[%d].kind обязательно", i)
		}
		switch src.Kind {
		case "postgres":
			if src.Config.DSNEnv == "" {
				return fmt.Errorf("поле sources[%d].config.dsn_env обязательно для postgres", i)
			}
		case "files":
			if src.Config.Path == "" {
				return fmt.Errorf("поле sources[%d].config.path обязательно для files", i)
			}
			if src.Config.MaxDepth < 0 {
				return fmt.Errorf("поле sources[%d].config.max_depth не может быть отрицательным", i)
			}
		case "rest":
			if src.Config.BaseURL == "" {
				return fmt.Errorf("поле sources[%d].config.base_url обязательно для rest", i)
			}
			if src.Config.Discovery == nil {
				return fmt.Errorf("поле sources[%d].config.discovery обязательно для rest", i)
			}
			if src.Config.Discovery.Mode != "openapi" {
				return fmt.Errorf("поле sources[%d].config.discovery.mode должно быть равно openapi", i)
			}
			if src.Config.Discovery.OpenAPIURL == "" {
				return fmt.Errorf("поле sources[%d].config.discovery.openapi_url обязательно для rest", i)
			}
		default:
			return fmt.Errorf("неподдерживаемый тип источника: %s", src.Kind)
		}
	}
	return nil
}
