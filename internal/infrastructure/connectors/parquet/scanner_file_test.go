package parquet

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/EdOoO21/metadata-parser/internal/domain/types"
	"github.com/EdOoO21/metadata-parser/internal/settings"
	pq "github.com/parquet-go/parquet-go"
)

type parquetPerson struct {
	ID        int64  `parquet:"id"`
	Name      string `parquet:"name"`
	CreatedAt string `parquet:"created_at"`
}

func TestScanParquetFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "people.parquet")
	writeParquetFile(t, path, []parquetPerson{
		{ID: 1, Name: "Alice", CreatedAt: "2026-03-28T10:00:00Z"},
		{ID: 2, Name: "Bob", CreatedAt: "2026-03-29T10:00:00Z"},
	})

	dataset, err := scanParquetFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dataset.Dataset.ProfileStatus != types.ProfileStatusProfiled {
		t.Fatalf("unexpected profile status: %s", dataset.Dataset.ProfileStatus)
	}
	if dataset.Dataset.RowCount == nil || *dataset.Dataset.RowCount != 2 {
		t.Fatalf("unexpected row count: %+v", dataset.Dataset.RowCount)
	}
	if len(dataset.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(dataset.Columns))
	}
	if dataset.Columns[0].Stat == nil {
		t.Fatal("expected column stats")
	}
}

func TestScannerParseSource_ParquetFileAndErrorCases(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "people.parquet")
	writeParquetFile(t, path, []parquetPerson{{ID: 1, Name: "Alice", CreatedAt: "2026-03-28T10:00:00Z"}})

	scanner := NewScanner()
	result, err := scanner.ParseSource(context.Background(), settings.SourceConfig{
		Name: "demo-parquet",
		Kind: "files",
		Config: settings.SourceConfigDetails{
			Path: path,
		},
	})
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(result.Datasets) != 1 {
		t.Fatalf("expected 1 dataset, got %d", len(result.Datasets))
	}

	_, err = scanner.ParseSource(context.Background(), settings.SourceConfig{
		Name: "demo-parquet",
		Kind: "files",
		Config: settings.SourceConfigDetails{
			Path: filepath.Join(t.TempDir(), "missing"),
		},
	})
	if err == nil {
		t.Fatal("expected missing path error")
	}

	badPath := filepath.Join(t.TempDir(), "bad.parquet")
	if err := os.WriteFile(badPath, []byte("not parquet"), 0o644); err != nil {
		t.Fatalf("write bad parquet: %v", err)
	}
	_, err = scanner.ParseSource(context.Background(), settings.SourceConfig{
		Name: "demo-parquet",
		Kind: "files",
		Config: settings.SourceConfigDetails{
			Path: badPath,
		},
	})
	if err == nil {
		t.Fatal("expected invalid parquet error")
	}
}

func writeParquetFile(t *testing.T, path string, rows []parquetPerson) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create parquet file: %v", err)
	}

	writer := pq.NewGenericWriter[parquetPerson](file)
	if _, err := writer.Write(rows); err != nil {
		_ = file.Close()
		t.Fatalf("write parquet rows: %v", err)
	}
	if err := writer.Close(); err != nil {
		_ = file.Close()
		t.Fatalf("close writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close file: %v", err)
	}
}
