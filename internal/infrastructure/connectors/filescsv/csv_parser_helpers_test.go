package filescsv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeCSVParseOptions_Defaults(t *testing.T) {
	t.Parallel()

	opts := normalizeCSVParseOptions(CSVParseOptions{})

	if opts.Delimiter != ',' {
		t.Fatalf("unexpected delimiter: %q", opts.Delimiter)
	}
	if opts.GeneratedColumnPrefix != "column_" {
		t.Fatalf("unexpected generated prefix: %q", opts.GeneratedColumnPrefix)
	}
}

func TestResolveLocalPath_FileURL(t *testing.T) {
	t.Parallel()

	got, err := resolveLocalPath("file:///tmp/demo.csv")
	if err != nil {
		t.Fatalf("resolveLocalPath returned error: %v", err)
	}

	want := filepath.Clean("/tmp/demo.csv")
	if got != want {
		t.Fatalf("unexpected resolved path: got %q want %q", got, want)
	}
}

func TestValidateHeaders_Duplicate(t *testing.T) {
	t.Parallel()

	err := validateHeaders([]string{"id", "ID"})
	if err == nil {
		t.Fatalf("expected duplicate header error")
	}
}

func TestCloneStringMap_ReturnsCopy(t *testing.T) {
	t.Parallel()

	src := map[string]string{"name": "Alice"}
	got := cloneStringMap(src)
	got["name"] = "Bob"

	if src["name"] != "Alice" {
		t.Fatalf("expected source map to stay unchanged, got %q", src["name"])
	}
}

func TestDiscoverCSVPaths_RespectsDepth(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	deeper := filepath.Join(nested, "deeper")

	if err := os.MkdirAll(deeper, 0o755); err != nil {
		t.Fatalf("mkdir nested dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "root.csv"), []byte("id\n1\n"), 0o644); err != nil {
		t.Fatalf("write root csv: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nested, "nested.csv"), []byte("id\n2\n"), 0o644); err != nil {
		t.Fatalf("write nested csv: %v", err)
	}
	if err := os.WriteFile(filepath.Join(deeper, "deep.csv"), []byte("id\n3\n"), 0o644); err != nil {
		t.Fatalf("write deep csv: %v", err)
	}

	paths, err := discoverCSVPaths(root, 1)
	if err != nil {
		t.Fatalf("discoverCSVPaths returned error: %v", err)
	}

	if len(paths) != 2 {
		t.Fatalf("expected 2 csv files with depth=1, got %d", len(paths))
	}
}
