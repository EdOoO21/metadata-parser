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
	if err := validateConfig(cfg); err == nil {
		t.Fatal("expected missing catalog error")
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
