package filescsv

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EdOoO21/metadata-parser/internal/domain/types"
	"github.com/EdOoO21/metadata-parser/internal/settings"
)

func TestCSVParser_Parse_WithHeader(t *testing.T) {
	t.Parallel()

	input := "id,name\n1,Alice\n2,Bob\n"

	parser := NewCSVParser()
	result, err := parser.Parse(context.Background(), strings.NewReader(input), DefaultCSVParseOptions())
	if err != nil {
		t.Fatalf("parse returned error: %v", err)
	}

	if len(result.Headers) != 2 {
		t.Fatalf("unexpected headers len: %d", len(result.Headers))
	}
	if result.Headers[0] != "id" || result.Headers[1] != "name" {
		t.Fatalf("unexpected headers: %#v", result.Headers)
	}
	if len(result.Rows) != 2 {
		t.Fatalf("unexpected rows len: %d", len(result.Rows))
	}
	if result.Rows[0].Values["id"] != "1" || result.Rows[0].Values["name"] != "Alice" {
		t.Fatalf("unexpected first row: %#v", result.Rows[0].Values)
	}
}

func TestCSVParser_Parse_WithoutHeader(t *testing.T) {
	t.Parallel()

	input := "1,Alice\n2,Bob\n"

	parser := NewCSVParser()
	opts := DefaultCSVParseOptions()
	opts.HasHeaderRecord = false

	result, err := parser.Parse(context.Background(), strings.NewReader(input), opts)
	if err != nil {
		t.Fatalf("parse returned error: %v", err)
	}

	if len(result.Headers) != 2 {
		t.Fatalf("unexpected headers len: %d", len(result.Headers))
	}
	if result.Headers[0] != "column_1" || result.Headers[1] != "column_2" {
		t.Fatalf("unexpected generated headers: %#v", result.Headers)
	}
	if result.Rows[1].Values["column_2"] != "Bob" {
		t.Fatalf("unexpected second row: %#v", result.Rows[1].Values)
	}
}

func TestCSVParser_Parse_DuplicateHeaders(t *testing.T) {
	t.Parallel()

	input := "id,id\n1,Alice\n"

	parser := NewCSVParser()
	_, err := parser.Parse(context.Background(), strings.NewReader(input), DefaultCSVParseOptions())
	if err == nil {
		t.Fatalf("expected duplicate headers error")
	}
}

func TestCSVParser_ParseSource(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "people.csv")

	err := os.WriteFile(path, []byte("id,name\n1,Alice\n2,Bob\n"), 0o644)
	if err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	parser := NewCSVParser()
	result, err := parser.ParseSource(context.Background(), settings.SourceConfig{
		Name: "people",
		Kind: "files",
		Config: settings.SourceConfigDetails{
			Path: path,
		},
	})
	if err != nil {
		t.Fatalf("parse source returned error: %v", err)
	}

	if len(result.Datasets) != 1 {
		t.Fatalf("expected one dataset, got %d", len(result.Datasets))
	}

	dataset := result.Datasets[0]
	if dataset.Dataset.Name != "people.csv" {
		t.Fatalf("unexpected dataset name: %q", dataset.Dataset.Name)
	}
	if len(dataset.Columns) != 2 {
		t.Fatalf("unexpected column count: %d", len(dataset.Columns))
	}
	if dataset.Columns[1].Column.Name != "name" {
		t.Fatalf("unexpected second column: %q", dataset.Columns[1].Column.Name)
	}
	if dataset.Dataset.RowCount == nil || *dataset.Dataset.RowCount != 2 {
		t.Fatalf("unexpected row count: %#v", dataset.Dataset.RowCount)
	}
	if dataset.Columns[0].Column.NormalizedType != types.CanonicalTypeNumber {
		t.Fatalf("unexpected first column normalized type: %q", dataset.Columns[0].Column.NormalizedType)
	}
	if dataset.Columns[1].Column.NormalizedType != types.CanonicalTypeString {
		t.Fatalf("unexpected second column normalized type: %q", dataset.Columns[1].Column.NormalizedType)
	}
	if dataset.Columns[1].Stat == nil || dataset.Columns[1].Stat.NonNullCount != 2 {
		t.Fatalf("unexpected second column stat: %#v", dataset.Columns[1].Stat)
	}
	if len(dataset.Columns[1].TopValues) != 2 {
		t.Fatalf("unexpected top values len: %d", len(dataset.Columns[1].TopValues))
	}
	var topName string
	if err := json.Unmarshal(dataset.Columns[1].TopValues[0].ValueJSON, &topName); err != nil {
		t.Fatalf("unmarshal top value: %v", err)
	}
	if topName != "Alice" {
		t.Fatalf("unexpected top value: %q", topName)
	}
}

func TestCSVParser_ParseSource_DirectoryProfilesAllCSVFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "orders.csv"), []byte("id,price,active\n1,10.5,true\n2,11.0,false\n"), 0o644)
	if err != nil {
		t.Fatalf("write orders file: %v", err)
	}

	err = os.WriteFile(filepath.Join(dir, "users.csv"), []byte("id,created_at\n1,2024-01-01T10:00:00Z\n2,2024-01-03T10:00:00Z\n"), 0o644)
	if err != nil {
		t.Fatalf("write users file: %v", err)
	}

	parser := NewCSVParser()
	result, err := parser.ParseSource(context.Background(), settings.SourceConfig{
		Name: "demo-files",
		Kind: "files",
		Config: settings.SourceConfigDetails{
			Path:     dir,
			MaxDepth: 0,
		},
	})
	if err != nil {
		t.Fatalf("parse source returned error: %v", err)
	}

	if len(result.Datasets) != 2 {
		t.Fatalf("expected two datasets, got %d", len(result.Datasets))
	}

	first := result.Datasets[0]
	second := result.Datasets[1]
	if first.Dataset.Name != "orders.csv" || second.Dataset.Name != "users.csv" {
		t.Fatalf("unexpected dataset names: %q, %q", first.Dataset.Name, second.Dataset.Name)
	}

	if first.Columns[1].Column.NormalizedType != types.CanonicalTypeNumber {
		t.Fatalf("unexpected price normalized type: %q", first.Columns[1].Column.NormalizedType)
	}
	if first.Columns[2].Column.NormalizedType != types.CanonicalTypeBoolean {
		t.Fatalf("unexpected active normalized type: %q", first.Columns[2].Column.NormalizedType)
	}
	if second.Columns[1].Column.NormalizedType != types.CanonicalTypeTimestamp {
		t.Fatalf("unexpected created_at normalized type: %q", second.Columns[1].Column.NormalizedType)
	}
}
