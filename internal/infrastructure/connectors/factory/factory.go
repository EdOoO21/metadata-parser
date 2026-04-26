package factory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/EdOoO21/metadata-parser/internal/application/contracts"
	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"
	"github.com/EdOoO21/metadata-parser/internal/settings"
)

type Factory struct {
	csvScanner      appports.SourceScanner
	parquetScanner  appports.SourceScanner
	postgresScanner appports.SourceScanner
	restScanner     appports.SourceScanner
}

func New(
	csvScanner appports.SourceScanner,
	parquetScanner appports.SourceScanner,
	postgresScanner appports.SourceScanner,
	restScanner appports.SourceScanner,
) *Factory {
	return &Factory{
		csvScanner:      csvScanner,
		parquetScanner:  parquetScanner,
		postgresScanner: postgresScanner,
		restScanner:     restScanner,
	}
}

func (f *Factory) ForSource(src settings.SourceConfig) (appports.SourceScanner, error) {
	switch src.Kind {
	case string(types.SourceKindPostgres):
		if f.postgresScanner == nil {
			return nil, fmt.Errorf("postgres scanner is not configured")
		}
		return f.postgresScanner, nil
	case string(types.SourceKindREST):
		if f.restScanner == nil {
			return nil, fmt.Errorf("rest scanner is not configured")
		}
		return f.restScanner, nil
	case string(types.SourceKindFiles):
		return f.fileScannerForSource(src)
	default:
		return nil, fmt.Errorf("unsupported source kind %q", src.Kind)
	}
}

func (f *Factory) fileScannerForSource(src settings.SourceConfig) (appports.SourceScanner, error) {
	path, err := resolveFilesPath(src.Config.Path)
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, fmt.Errorf("files source path is empty")
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat files source path: %w", err)
	}

	if !info.IsDir() {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".csv":
			if f.csvScanner == nil {
				return nil, fmt.Errorf("csv scanner is not configured")
			}
			return f.csvScanner, nil
		case ".parquet":
			if f.parquetScanner == nil {
				return nil, fmt.Errorf("parquet scanner is not configured")
			}
			return f.parquetScanner, nil
		default:
			return nil, fmt.Errorf("unsupported files source extension %q", strings.ToLower(filepath.Ext(path)))
		}
	}

	parts := make([]appports.SourceScanner, 0, 2)
	if f.csvScanner != nil {
		parts = append(parts, scopedFilesScanner{scanner: f.csvScanner, extension: ".csv"})
	}
	if f.parquetScanner != nil {
		parts = append(parts, scopedFilesScanner{scanner: f.parquetScanner, extension: ".parquet"})
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("no file scanners are configured")
	}

	return compositeScanner{
		sourceKind: types.SourceKindFiles,
		scanners:   parts,
	}, nil
}

func resolveFilesPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}

	if strings.HasPrefix(strings.ToLower(path), "file://") {
		u, err := url.Parse(path)
		if err != nil {
			return "", fmt.Errorf("parse file url: %w", err)
		}
		path = u.Path
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}

	return abs, nil
}

type compositeScanner struct {
	sourceKind types.SourceKind
	scanners   []appports.SourceScanner
}

func (s compositeScanner) ParseSource(ctx context.Context, src settings.SourceConfig) (*contracts.SourceScanResult, error) {
	effectiveConfigJSON, err := json.Marshal(src.Config)
	if err != nil {
		return nil, fmt.Errorf("marshal effective config: %w", err)
	}

	result := &contracts.SourceScanResult{
		Source: model.Source{
			Name: src.Name,
			Kind: s.sourceKind,
		},
		EffectiveConfigJSON: effectiveConfigJSON,
		Datasets:            []contracts.ScannedDataset{},
	}

	for _, scanner := range s.scanners {
		partial, err := scanner.ParseSource(ctx, src)
		if err != nil {
			continue
		}
		result.Datasets = append(result.Datasets, partial.Datasets...)
	}

	if len(result.Datasets) == 0 {
		return nil, fmt.Errorf("files source %q did not contain supported datasets", src.Name)
	}

	return result, nil
}

type scopedFilesScanner struct {
	scanner   appports.SourceScanner
	extension string
}

func (s scopedFilesScanner) ParseSource(ctx context.Context, src settings.SourceConfig) (*contracts.SourceScanResult, error) {
	path, err := resolveFilesPath(src.Config.Path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() && !strings.EqualFold(filepath.Ext(path), s.extension) {
		return nil, fmt.Errorf("source path does not match %s", s.extension)
	}
	return s.scanner.ParseSource(ctx, src)
}

var _ appports.SourceScannerFactory = (*Factory)(nil)
var _ appports.SourceScanner = compositeScanner{}
