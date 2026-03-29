package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFileIfPresent_SetsMissingEnvVars(t *testing.T) {
	t.Setenv("ALREADY_SET", "keep")

	path := filepath.Join(t.TempDir(), ".env")
	content := "" +
		"# comment\n" +
		"CATALOG_DSN=\"postgres://localhost:5432/catalog\"\n" +
		"export DEMO_PG_DSN='postgres://localhost:55433/source_case_1'\n" +
		"ALREADY_SET=override\n"

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	if err := LoadEnvFileIfPresent(path); err != nil {
		t.Fatalf("load env: %v", err)
	}

	if got := os.Getenv("CATALOG_DSN"); got != "postgres://localhost:5432/catalog" {
		t.Fatalf("unexpected CATALOG_DSN: %q", got)
	}

	if got := os.Getenv("DEMO_PG_DSN"); got != "postgres://localhost:55433/source_case_1" {
		t.Fatalf("unexpected DEMO_PG_DSN: %q", got)
	}

	if got := os.Getenv("ALREADY_SET"); got != "keep" {
		t.Fatalf("expected existing env to be preserved, got %q", got)
	}
}

func TestLoadEnvFileIfPresent_MissingFile(t *testing.T) {
	if err := LoadEnvFileIfPresent(filepath.Join(t.TempDir(), ".env")); err != nil {
		t.Fatalf("expected missing env file to be ignored, got %v", err)
	}
}

func TestLoadEnvFileIfPresent_InvalidLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("BROKEN_LINE"), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	if err := LoadEnvFileIfPresent(path); err == nil {
		t.Fatal("expected parse error")
	}
}
