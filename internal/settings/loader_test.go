package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoaderLoad_Success(t *testing.T) {
	loader := NewLoader()
	cfg, err := loader.Load(filepath.Join("testdata", "valid_demo.yaml"))
	if err != nil {
		t.Fatalf("expected config to load successfully, got error: %v", err)
	}

	if cfg.Version != 1 {
		t.Fatalf("expected version 1, got %d", cfg.Version)
	}
	if len(cfg.Sources) != 3 {
		t.Fatalf("expected 3 sources, got %d", len(cfg.Sources))
	}
}

func TestLoaderLoad_DuplicateSourceName(t *testing.T) {
	loader := NewLoader()
	_, err := loader.Load(filepath.Join("testdata", "duplicate_source_name.yaml"))
	if err == nil {
		t.Fatal("expected duplicate source name validation error, got nil")
	}
}

func TestLoaderLoad_InvalidRestDiscovery(t *testing.T) {
	loader := NewLoader()
	_, err := loader.Load(filepath.Join("testdata", "invalid_rest_discovery.yaml"))
	if err == nil {
		t.Fatal("expected invalid rest discovery validation error, got nil")
	}
}

func TestLoaderLoad_InvalidFilesDepth(t *testing.T) {
	loader := NewLoader()
	_, err := loader.Load(filepath.Join("testdata", "invalid_files_depth.yaml"))
	if err == nil {
		t.Fatal("expected invalid files max_depth validation error, got nil")
	}
}

func TestLoaderLoad_FileNotFound(t *testing.T) {
	loader := NewLoader()
	_, err := loader.Load(filepath.Join("testdata", "missing.yaml"))
	if err == nil {
		t.Fatal("expected read config error, got nil")
	}
}

func TestValidateConfig_UnsupportedVersionAndNoSources(t *testing.T) {
	cfg := &AppConfig{
		Version: 2,
		Catalog: CatalogConfig{DSNEnv: "CATALOG_DSN"},
		Sources: []SourceConfig{{Name: "src", Kind: "files", Config: SourceConfigDetails{Path: "./demo"}}},
	}
	if err := validateConfig(cfg); err == nil {
		t.Fatal("expected unsupported version error")
	}

	cfg = &AppConfig{
		Version: 1,
		Catalog: CatalogConfig{DSNEnv: "CATALOG_DSN"},
	}
	if err := validateConfig(cfg); err == nil {
		t.Fatal("expected no sources error")
	}
}

func TestLoaderLoad_InvalidYAML(t *testing.T) {
	loader := NewLoader()
	path := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(path, []byte("version: ["), 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	if _, err := loader.Load(path); err == nil {
		t.Fatal("expected yaml parse error")
	}
}

func TestValidateConfig_MissingCatalogAndUnsupportedKind(t *testing.T) {
	cfg := &AppConfig{
		Version: 1,
		Sources: []SourceConfig{{Name: "src", Kind: "files", Config: SourceConfigDetails{Path: "./demo"}}},
	}
	if err := validateConfig(cfg); err != nil {
		t.Fatalf("expected default catalog env to be applied, got error: %v", err)
	}
	if cfg.Catalog.DSNEnv != defaultCatalogDSNEnv {
		t.Fatalf("expected default catalog env %q, got %q", defaultCatalogDSNEnv, cfg.Catalog.DSNEnv)
	}

	cfg = &AppConfig{
		Version: 1,
		Catalog: CatalogConfig{DSNEnv: "CATALOG_DSN"},
		Sources: []SourceConfig{{Name: "src", Kind: "kafka"}},
	}
	if err := validateConfig(cfg); err == nil {
		t.Fatal("expected unsupported kind error")
	}
}

func TestValidateConfig_PostgresMode(t *testing.T) {
	t.Parallel()

	cfg := &AppConfig{
		Version: 1,
		Catalog: CatalogConfig{DSNEnv: "CATALOG_DSN"},
		Sources: []SourceConfig{{
			Name: "pg",
			Kind: "postgres",
			Config: SourceConfigDetails{
				DSNEnv: "DEMO_PG_DSN",
				Mode:   "sampled",
			},
		}},
	}
	if err := validateConfig(cfg); err != nil {
		t.Fatalf("expected sampled mode to be valid, got error: %v", err)
	}

	cfg.Sources[0].Config.Mode = "schema_only"
	if err := validateConfig(cfg); err != nil {
		t.Fatalf("expected schema_only mode to be valid, got error: %v", err)
	}

	cfg.Sources[0].Config.Mode = "fast"
	if err := validateConfig(cfg); err == nil {
		t.Fatal("expected invalid postgres mode error")
	}
}

func TestValidateConfig_RestMaxResponseBytes(t *testing.T) {
	t.Parallel()

	cfg := &AppConfig{
		Version: 1,
		Catalog: CatalogConfig{DSNEnv: "CATALOG_DSN"},
		Sources: []SourceConfig{{
			Name: "api",
			Kind: "rest",
			Config: SourceConfigDetails{
				BaseURL:          "http://localhost:8080",
				MaxResponseBytes: 1024,
				Discovery: &DiscoveryConfig{
					Mode:       "openapi",
					OpenAPIURL: "http://localhost:8080/openapi.json",
				},
			},
		}},
	}
	if err := validateConfig(cfg); err != nil {
		t.Fatalf("expected positive max_response_bytes to be valid, got error: %v", err)
	}

	cfg.Sources[0].Config.MaxResponseBytes = -1
	if err := validateConfig(cfg); err == nil {
		t.Fatal("expected negative max_response_bytes validation error")
	}
}
