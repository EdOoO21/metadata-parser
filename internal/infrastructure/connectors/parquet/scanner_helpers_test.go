package parquet

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiscoverParquetPaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	topFile := filepath.Join(root, "a.parquet")
	nestedDir := filepath.Join(root, "nested")
	deepDir := filepath.Join(nestedDir, "deep")

	if err := os.MkdirAll(deepDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for _, path := range []string{
		topFile,
		filepath.Join(nestedDir, "b.parquet"),
		filepath.Join(deepDir, "c.parquet"),
	} {
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	paths, err := discoverParquetPaths(root, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %+v", len(paths), paths)
	}

	if _, err := discoverParquetPaths(filepath.Join(root, "missing"), 1); err == nil {
		t.Fatal("expected missing path error")
	}

	txtPath := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(txtPath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write txt: %v", err)
	}
	if _, err := discoverParquetPaths(txtPath, 1); err == nil {
		t.Fatal("expected unsupported extension error")
	}

	emptyDir := filepath.Join(root, "empty")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatalf("mkdir empty: %v", err)
	}
	if _, err := discoverParquetPaths(emptyDir, 1); err == nil {
		t.Fatal("expected no parquet files error")
	}
}

func TestPathDepthAndParquetPath(t *testing.T) {
	t.Parallel()

	if got := pathDepth("."); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
	if got := pathDepth(filepath.Join("a", "b", "c.parquet")); got != 2 {
		t.Fatalf("expected 2, got %d", got)
	}
	if !isParquetPath("/tmp/test.PARQUET") {
		t.Fatal("expected parquet path")
	}
}

func TestFlattenParquetRow_ArrayValues(t *testing.T) {
	t.Parallel()

	out := map[string]any{}
	flattenParquetRow("", map[string]any{
		"profile": map[string]any{
			"name": "Ivan",
		},
		"tags": []any{"a", "b"},
	}, out)

	if out["profile.name"] != "Ivan" {
		t.Fatalf("unexpected flattened value: %+v", out)
	}
	if _, ok := out["tags"]; !ok {
		t.Fatalf("expected tags in flattened output: %+v", out)
	}
}

func TestParquetConversionHelpers(t *testing.T) {
	t.Parallel()

	if !looksIntegerType("UINT64") {
		t.Fatal("expected integer type")
	}
	if value, ok := asFloat64(json.Number("12.5")); !ok || value != 12.5 {
		t.Fatalf("unexpected float conversion: %v %v", value, ok)
	}
	if _, ok := asFloat64(struct{}{}); ok {
		t.Fatal("expected unsupported float conversion")
	}
	if ts, ok := asTimestamp("2026-03-28"); !ok || ts.IsZero() {
		t.Fatalf("unexpected timestamp conversion: %v %v", ts, ok)
	}
	if _, ok := asTimestamp("not-a-date"); ok {
		t.Fatal("expected invalid timestamp conversion")
	}
}

func TestMarshalParquetNumberBounds(t *testing.T) {
	t.Parallel()

	minJSON, maxJSON, err := marshalParquetNumberBounds(1, 3, "int64")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(minJSON) != "1" || string(maxJSON) != "3" {
		t.Fatalf("unexpected integer bounds: %s %s", string(minJSON), string(maxJSON))
	}

	minJSON, maxJSON, err = marshalParquetNumberBounds(1.5, 3.5, "double")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(minJSON) != "1.5" || string(maxJSON) != "3.5" {
		t.Fatalf("unexpected float bounds: %s %s", string(minJSON), string(maxJSON))
	}
}

func TestSortParquetTopValues(t *testing.T) {
	t.Parallel()

	items := []topValueItem{
		{valueJSON: `"b"`, count: 1},
		{valueJSON: `"a"`, count: 2},
		{valueJSON: `"c"`, count: 2},
	}
	sortParquetTopValues(items)

	if items[0].valueJSON != `"a"` || items[1].valueJSON != `"c"` || items[2].valueJSON != `"b"` {
		t.Fatalf("unexpected sorted items: %+v", items)
	}
}

func TestAsTimestampTimeValue(t *testing.T) {
	t.Parallel()

	now := time.Now()
	got, ok := asTimestamp(now)
	if !ok || !got.Equal(now) {
		t.Fatalf("unexpected time conversion: %v %v", got, ok)
	}
}
