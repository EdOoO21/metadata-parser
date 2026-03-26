package settings

type AppConfig struct {
	Version int            `yaml:"version"`
	Catalog CatalogConfig  `yaml:"catalog"`
	Sources []SourceConfig `yaml:"sources"`
}

type CatalogConfig struct {
	DSNEnv string `yaml:"dsn_env"`
}

type SourceConfig struct {
	Name   string              `yaml:"name"`
	Kind   string              `yaml:"kind"`
	Config SourceConfigDetails `yaml:"config"`
}

type SourceConfigDetails struct {
	DSNEnv    string           `yaml:"dsn_env,omitempty"`
	Path      string           `yaml:"path,omitempty"`
	MaxDepth  int              `yaml:"max_depth,omitempty"`
	BaseURL   string           `yaml:"base_url,omitempty"`
	Discovery *DiscoveryConfig `yaml:"discovery,omitempty"`
}

type DiscoveryConfig struct {
	Mode       string `yaml:"mode"`
	OpenAPIURL string `yaml:"openapi_url"`
}
