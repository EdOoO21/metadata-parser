package factory

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/EdOoO21/metadata-parser/internal/application/contracts"
	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
	"github.com/EdOoO21/metadata-parser/internal/settings"
)

type stubScanner struct {
	result *contracts.SourceScanResult
	err    error
	calls  int
}

func (s *stubScanner) ParseSource(_ context.Context, _ settings.SourceConfig) (*contracts.SourceScanResult, error) {
	s.calls++
	return s.result, s.err
}

func TestFactoryForSource_ReturnsCSVScannerForCSVFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "people.csv")
	if err := os.WriteFile(csvPath, []byte("id,name\n1,Alice\n"), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	csvScanner := &stubScanner{}
	f := New(csvScanner, nil, nil, nil)

	scanner, err := f.ForSource(settings.SourceConfig{
		Name: "people",
		Kind: "files",
		Config: settings.SourceConfigDetails{
			Path: csvPath,
		},
	})
	if err != nil {
		t.Fatalf("ForSource returned error: %v", err)
	}
	if scanner != csvScanner {
		t.Fatalf("expected csv scanner, got %T", scanner)
	}
}

func TestFactoryForSource_ReturnsParquetScannerForParquetFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	parquetPath := filepath.Join(tmpDir, "events.parquet")
	if err := os.WriteFile(parquetPath, []byte("PAR1"), 0o644); err != nil {
		t.Fatalf("write parquet placeholder: %v", err)
	}

	parquetScanner := &stubScanner{}
	f := New(nil, parquetScanner, nil, nil)

	scanner, err := f.ForSource(settings.SourceConfig{
		Name: "events",
		Kind: "files",
		Config: settings.SourceConfigDetails{
			Path: parquetPath,
		},
	})
	if err != nil {
		t.Fatalf("ForSource returned error: %v", err)
	}
	if scanner != parquetScanner {
		t.Fatalf("expected parquet scanner, got %T", scanner)
	}
}

func TestFactoryForSource_ReturnsCompositeScannerForDirectory(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "people.csv")
	parquetPath := filepath.Join(tmpDir, "events.parquet")
	if err := os.WriteFile(csvPath, []byte("id\n1\n"), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}
	if err := os.WriteFile(parquetPath, []byte("PAR1"), 0o644); err != nil {
		t.Fatalf("write parquet placeholder: %v", err)
	}

	csvScanner := &stubScanner{
		result: &contracts.SourceScanResult{
			Datasets: []contracts.ScannedDataset{{}},
		},
	}
	parquetScanner := &stubScanner{
		result: &contracts.SourceScanResult{
			Datasets: []contracts.ScannedDataset{{}},
		},
	}
	f := New(csvScanner, parquetScanner, nil, nil)

	scanner, err := f.ForSource(settings.SourceConfig{
		Name: "files-source",
		Kind: "files",
		Config: settings.SourceConfigDetails{
			Path: tmpDir,
		},
	})
	if err != nil {
		t.Fatalf("ForSource returned error: %v", err)
	}

	result, err := scanner.ParseSource(context.Background(), settings.SourceConfig{
		Name: "files-source",
		Kind: "files",
		Config: settings.SourceConfigDetails{
			Path: tmpDir,
		},
	})
	if err != nil {
		t.Fatalf("ParseSource returned error: %v", err)
	}
	if got := len(result.Datasets); got != 2 {
		t.Fatalf("expected 2 datasets, got %d", got)
	}
	if csvScanner.calls != 1 {
		t.Fatalf("expected csv scanner to be called once, got %d", csvScanner.calls)
	}
	if parquetScanner.calls != 1 {
		t.Fatalf("expected parquet scanner to be called once, got %d", parquetScanner.calls)
	}
}

func TestFactoryForSource_ReturnsConfiguredNetworkScanners(t *testing.T) {
	t.Parallel()

	postgresScanner := &stubScanner{}
	restScanner := &stubScanner{}
	f := New(nil, nil, postgresScanner, restScanner)

	gotPostgres, err := f.ForSource(settings.SourceConfig{Name: "pg", Kind: "postgres"})
	if err != nil {
		t.Fatalf("postgres ForSource returned error: %v", err)
	}
	if gotPostgres != postgresScanner {
		t.Fatalf("expected postgres scanner, got %T", gotPostgres)
	}

	gotREST, err := f.ForSource(settings.SourceConfig{Name: "api", Kind: "rest"})
	if err != nil {
		t.Fatalf("rest ForSource returned error: %v", err)
	}
	if gotREST != restScanner {
		t.Fatalf("expected rest scanner, got %T", gotREST)
	}
}

func TestFactoryForSource_ErrorCases(t *testing.T) {
	t.Parallel()

	f := New(nil, nil, nil, nil)

	if _, err := f.ForSource(settings.SourceConfig{Name: "pg", Kind: "postgres"}); err == nil {
		t.Fatal("expected missing postgres scanner error")
	}
	if _, err := f.ForSource(settings.SourceConfig{Name: "api", Kind: "rest"}); err == nil {
		t.Fatal("expected missing rest scanner error")
	}
	if _, err := f.ForSource(settings.SourceConfig{Name: "x", Kind: "unknown"}); err == nil {
		t.Fatal("expected unsupported kind error")
	}

	tmpDir := t.TempDir()
	txtPath := filepath.Join(tmpDir, "notes.txt")
	if err := os.WriteFile(txtPath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write txt: %v", err)
	}
	if _, err := f.ForSource(settings.SourceConfig{
		Name: "txt",
		Kind: "files",
		Config: settings.SourceConfigDetails{
			Path: txtPath,
		},
	}); err == nil {
		t.Fatal("expected unsupported extension error")
	}

	if _, err := f.ForSource(settings.SourceConfig{
		Name: "dir",
		Kind: "files",
		Config: settings.SourceConfigDetails{
			Path: tmpDir,
		},
	}); err == nil {
		t.Fatal("expected no file scanners configured error")
	}
}

func TestResolveFilesPathAndScopedScanner(t *testing.T) {
	t.Parallel()

	resolved, err := resolveFilesPath("")
	if err != nil || resolved != "" {
		t.Fatalf("expected empty path resolution, got %q %v", resolved, err)
	}

	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "people.csv")
	if err := os.WriteFile(csvPath, []byte("id\n1\n"), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	resolved, err = resolveFilesPath("file://" + csvPath)
	if err != nil {
		t.Fatalf("resolve file url: %v", err)
	}
	if resolved == "" {
		t.Fatal("expected resolved path")
	}

	inner := &stubScanner{result: &contracts.SourceScanResult{Datasets: []contracts.ScannedDataset{{}}}}
	scanner := scopedFilesScanner{scanner: inner, extension: ".csv"}
	if _, err := scanner.ParseSource(context.Background(), settings.SourceConfig{
		Name: "csv",
		Kind: "files",
		Config: settings.SourceConfigDetails{
			Path: csvPath,
		},
	}); err != nil {
		t.Fatalf("unexpected scoped scanner error: %v", err)
	}

	if _, err := scanner.ParseSource(context.Background(), settings.SourceConfig{
		Name: "csv",
		Kind: "files",
		Config: settings.SourceConfigDetails{
			Path: tmpDir,
		},
	}); err != nil {
		t.Fatalf("unexpected directory scoped scanner error: %v", err)
	}

	parquetPath := filepath.Join(tmpDir, "events.parquet")
	if err := os.WriteFile(parquetPath, []byte("PAR1"), 0o644); err != nil {
		t.Fatalf("write parquet: %v", err)
	}
	if _, err := scanner.ParseSource(context.Background(), settings.SourceConfig{
		Name: "csv",
		Kind: "files",
		Config: settings.SourceConfigDetails{
			Path: parquetPath,
		},
	}); err == nil {
		t.Fatal("expected extension mismatch error")
	}
}

func TestCompositeScanner_ErrorPaths(t *testing.T) {
	t.Parallel()

	scanner := compositeScanner{
		sourceKind: "files",
		scanners: []appports.SourceScanner{
			&stubScanner{err: os.ErrNotExist},
			&stubScanner{err: os.ErrPermission},
		},
	}
	if _, err := scanner.ParseSource(context.Background(), settings.SourceConfig{
		Name:   "files-source",
		Kind:   "files",
		Config: settings.SourceConfigDetails{Path: "."},
	}); err == nil {
		t.Fatal("expected no datasets error")
	}
}

func TestResolveFilesPath_InvalidFileURL(t *testing.T) {
	t.Parallel()

	if _, err := resolveFilesPath("file://%zz"); err == nil {
		t.Fatal("expected invalid file url error")
	}
}

var _ appports.SourceScanner = (*stubScanner)(nil)
