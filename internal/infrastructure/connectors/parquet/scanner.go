package parquet

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/EdOoO21/metadata-parser/internal/application/contracts"
	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"
	"github.com/EdOoO21/metadata-parser/internal/infrastructure/connectors/shared"
	"github.com/EdOoO21/metadata-parser/internal/settings"
	pq "github.com/parquet-go/parquet-go"
)

type Scanner struct{}

func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) ParseSource(ctx context.Context, src settings.SourceConfig) (*contracts.SourceScanResult, error) {
	_ = ctx

	effectiveConfigJSON, err := json.Marshal(src.Config)
	if err != nil {
		return nil, fmt.Errorf("marshal effective config: %w", err)
	}

	paths, err := discoverParquetPaths(src.Config.Path, src.Config.MaxDepth)
	if err != nil {
		return nil, err
	}

	datasets := make([]contracts.ScannedDataset, 0, len(paths))
	for _, path := range paths {
		dataset, err := scanParquetFile(path)
		if err != nil {
			return nil, fmt.Errorf("scan parquet file %q: %w", path, err)
		}
		datasets = append(datasets, dataset)
	}

	return &contracts.SourceScanResult{
		Source: model.Source{
			Name: src.Name,
			Kind: types.SourceKindFiles,
		},
		EffectiveConfigJSON: effectiveConfigJSON,
		Datasets:            datasets,
	}, nil
}

func scanParquetFile(path string) (contracts.ScannedDataset, error) {
	file, err := os.Open(path)
	if err != nil {
		return contracts.ScannedDataset{}, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return contracts.ScannedDataset{}, fmt.Errorf("stat file: %w", err)
	}

	parquetFile, err := pq.OpenFile(file, info.Size())
	if err != nil {
		return contracts.ScannedDataset{}, fmt.Errorf("open parquet footer: %w", err)
	}

	metadataJSON, err := json.Marshal(map[string]any{
		"format":      "parquet",
		"source_path": path,
	})
	if err != nil {
		return contracts.ScannedDataset{}, fmt.Errorf("marshal metadata: %w", err)
	}

	rowCount := parquetFile.NumRows()
	columns := flattenParquetFields(parquetFile.Schema().Fields(), "", 0)
	if err := profileParquetColumns(file, columns); err != nil {
		errMessage := err.Error()
		return contracts.ScannedDataset{
			Dataset: model.Dataset{
				Kind:          types.DatasetKindFile,
				DatasetKey:    path,
				Name:          filepath.Base(path),
				Location:      path,
				RowCount:      &rowCount,
				DiscoveredAt:  time.Now().UTC(),
				ProfileStatus: types.ProfileStatusFailed,
				ProfileError:  &errMessage,
				MetadataJSON:  metadataJSON,
			},
			Columns: columns,
		}, nil
	}

	return contracts.ScannedDataset{
		Dataset: model.Dataset{
			Kind:          types.DatasetKindFile,
			DatasetKey:    path,
			Name:          filepath.Base(path),
			Location:      path,
			RowCount:      &rowCount,
			DiscoveredAt:  time.Now().UTC(),
			ProfileStatus: types.ProfileStatusProfiled,
			MetadataJSON:  metadataJSON,
		},
		Columns: columns,
	}, nil
}

func flattenParquetFields(fields []pq.Field, prefix string, ordinalBase int) []contracts.ScannedColumn {
	columns := make([]contracts.ScannedColumn, 0)
	ordinal := ordinalBase

	for _, field := range fields {
		name := field.Name()
		if prefix != "" {
			name = prefix + "." + name
		}

		if field.Leaf() {
			ordinal++
			originalType := field.Type().String()
			columns = append(columns, contracts.ScannedColumn{
				Column: model.Column{
					Name:            name,
					OriginalType:    originalType,
					NormalizedType:  shared.NormalizeType(originalType),
					IsNullable:      !field.Required(),
					OrdinalPosition: ordinal,
				},
			})
			continue
		}

		nested := flattenParquetFields(field.Fields(), name, ordinal)
		if len(nested) == 0 {
			continue
		}
		ordinal = nested[len(nested)-1].Column.OrdinalPosition
		columns = append(columns, nested...)
	}

	return columns
}

func discoverParquetPaths(path string, maxDepth int) ([]string, error) {
	rootPath, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return nil, fmt.Errorf("resolve absolute path: %w", err)
	}

	info, err := os.Stat(rootPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path not found: %s", path)
		}
		return nil, fmt.Errorf("stat path: %w", err)
	}

	if !info.IsDir() {
		if !isParquetPath(rootPath) {
			return nil, fmt.Errorf("%w: unsupported file extension %q", shared.ErrNoMatchingFiles, strings.ToLower(filepath.Ext(rootPath)))
		}
		return []string{rootPath}, nil
	}

	paths := make([]string, 0)
	err = filepath.WalkDir(rootPath, func(currentPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(rootPath, currentPath)
		if err != nil {
			return err
		}

		depth := pathDepth(relPath)
		if d.IsDir() {
			if currentPath != rootPath && depth > maxDepth {
				return filepath.SkipDir
			}
			return nil
		}

		if depth > maxDepth {
			return nil
		}
		if isParquetPath(currentPath) {
			paths = append(paths, currentPath)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk directory: %w", err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("%w: no parquet files found under %s", shared.ErrNoMatchingFiles, rootPath)
	}

	return paths, nil
}

func pathDepth(relPath string) int {
	if relPath == "." || strings.TrimSpace(relPath) == "" {
		return 0
	}
	return strings.Count(filepath.Clean(relPath), string(filepath.Separator))
}

func isParquetPath(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".parquet")
}

var _ appports.SourceScanner = (*Scanner)(nil)
