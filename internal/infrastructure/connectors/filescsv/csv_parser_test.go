package filescsv

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
}
