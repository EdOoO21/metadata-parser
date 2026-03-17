package config

import (
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
