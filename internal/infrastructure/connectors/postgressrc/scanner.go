package postgressrc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/EdOoO21/metadata-parser/internal/application/contracts"
	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"
	"github.com/EdOoO21/metadata-parser/internal/infrastructure/connectors/shared"
	catalogpg "github.com/EdOoO21/metadata-parser/internal/infrastructure/db/postgres"
	"github.com/EdOoO21/metadata-parser/internal/settings"
)

type Scanner struct{}

func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) ParseSource(ctx context.Context, src settings.SourceConfig) (*contracts.SourceScanResult, error) {
	effectiveConfigJSON, err := json.Marshal(src.Config)
	if err != nil {
		return nil, fmt.Errorf("marshal effective config: %w", err)
	}

	pool, err := catalogpg.NewPoolFromEnv(ctx, src.Config.DSNEnv)
	if err != nil {
		return nil, fmt.Errorf("open postgres source: %w", err)
	}
	defer pool.Close()

	rows, err := pool.Query(ctx, postgresDiscoveryQuery)
	if err != nil {
		return nil, fmt.Errorf("query postgres schema: %w", err)
	}
	defer rows.Close()

	datasets := make(map[string]*contracts.ScannedDataset)
	order := make([]string, 0, 32)

	for rows.Next() {
		var schemaName string
		var datasetName string
		var datasetKind string
		var datasetComment *string
		var columnName string
		var columnType string
		var columnNullable bool
		var columnComment *string
		var ordinal int

		if err := rows.Scan(
			&schemaName,
			&datasetName,
			&datasetKind,
			&datasetComment,
			&columnName,
			&columnType,
			&columnNullable,
			&columnComment,
			&ordinal,
		); err != nil {
			return nil, fmt.Errorf("scan postgres schema row: %w", err)
		}

		key := schemaName + "." + datasetName
		if _, ok := datasets[key]; !ok {
			metadataJSON, err := json.Marshal(map[string]any{
				"schema": schemaName,
			})
			if err != nil {
				return nil, fmt.Errorf("marshal dataset metadata: %w", err)
			}

			order = append(order, key)
			datasets[key] = &contracts.ScannedDataset{
				Dataset: model.Dataset{
					Kind:          mapDatasetKind(datasetKind),
					DatasetKey:    key,
					Name:          datasetName,
					Location:      key,
					Comment:       datasetComment,
					DiscoveredAt:  time.Now().UTC(),
					ProfileStatus: types.ProfileStatusDiscoveredOnly,
					MetadataJSON:  metadataJSON,
				},
				Columns: nil,
			}
		}

		dataset := datasets[key]
		dataset.Columns = append(dataset.Columns, contracts.ScannedColumn{
			Column: model.Column{
				Name:            columnName,
				OriginalType:    columnType,
				NormalizedType:  shared.NormalizeType(columnType),
				IsNullable:      columnNullable,
				Comment:         columnComment,
				OrdinalPosition: ordinal,
			},
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate postgres schema rows: %w", err)
	}

	scannedDatasets := make([]contracts.ScannedDataset, 0, len(order))
	for _, key := range order {
		dataset := datasets[key]
		if err := profileDataset(ctx, pool, dataset); err != nil {
			errMessage := err.Error()
			dataset.Dataset.ProfileStatus = types.ProfileStatusFailed
			dataset.Dataset.ProfileError = &errMessage
		} else {
			dataset.Dataset.ProfileStatus = types.ProfileStatusProfiled
			dataset.Dataset.ProfileError = nil
		}

		scannedDatasets = append(scannedDatasets, *dataset)
	}

	return &contracts.SourceScanResult{
		Source: model.Source{
			Name: src.Name,
			Kind: types.SourceKindPostgres,
		},
		EffectiveConfigJSON: effectiveConfigJSON,
		Datasets:            scannedDatasets,
	}, nil
}

func mapDatasetKind(value string) types.DatasetKind {
	switch value {
	case "VIEW":
		return types.DatasetKindView
	default:
		return types.DatasetKindTable
	}
}

const postgresDiscoveryQuery = `
SELECT
    n.nspname AS schema_name,
    c.relname AS dataset_name,
    c.relkind::text AS relkind,
    obj_description(c.oid, 'pg_class') AS dataset_comment,
    a.attname AS column_name,
    pg_catalog.format_type(a.atttypid, a.atttypmod) AS column_type,
    NOT a.attnotnull AS is_nullable,
    col_description(c.oid, a.attnum) AS column_comment,
    a.attnum AS ordinal_position
FROM pg_catalog.pg_class c
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
JOIN pg_catalog.pg_attribute a ON a.attrelid = c.oid
WHERE c.relkind IN ('r', 'v')
  AND n.nspname NOT IN ('pg_catalog', 'information_schema')
  AND a.attnum > 0
  AND NOT a.attisdropped
ORDER BY n.nspname, c.relname, a.attnum
`

var _ appports.SourceScanner = (*Scanner)(nil)
