package filescsv

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalFileSource_OpenRead(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sample.csv")

	err := os.WriteFile(path, []byte("id,name\n1,Alice\n"), 0o644)
	if err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	source := NewLocalFileSource()

	exists, err := source.Exists(context.Background(), LocalFile(path))
	if err != nil {
		t.Fatalf("exists returned error: %v", err)
	}
	if !exists {
		t.Fatalf("expected file to exist")
	}

	opened, err := source.OpenRead(context.Background(), LocalFile(path))
	if err != nil {
		t.Fatalf("open read returned error: %v", err)
	}
	defer opened.Stream.Close()

	if opened.Name != "sample.csv" {
		t.Fatalf("unexpected file name: %s", opened.Name)
	}

	if opened.ContentType != "text/csv" {
		t.Fatalf("unexpected content type: %s", opened.ContentType)
	}

	if opened.Size == nil || *opened.Size == 0 {
		t.Fatalf("expected non-zero file size")
	}
}
