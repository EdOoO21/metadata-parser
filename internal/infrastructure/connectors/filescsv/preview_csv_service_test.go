package filescsv

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPreviewCSVService_Execute(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "people.csv")

	err := os.WriteFile(path, []byte("id,name\n1,Alice\n2,Bob\n"), 0o644)
	if err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	service := NewPreviewCSVService(
		NewFileSourceResolver(NewLocalFileSource()),
		NewCSVConnector(),
	)

	result, err := service.Execute(context.Background(), LocalFile(path), 1)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if len(result.Headers) != 2 {
		t.Fatalf("unexpected headers len: %d", len(result.Headers))
	}
	if len(result.Rows) != 1 {
		t.Fatalf("expected preview to return only one row, got %d", len(result.Rows))
	}
	if result.Rows[0].Values["name"] != "Alice" {
		t.Fatalf("unexpected first row: %#v", result.Rows[0].Values)
	}
}
