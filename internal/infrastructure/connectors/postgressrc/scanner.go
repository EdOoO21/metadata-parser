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

type discoveryRow struct {
	schemaName     string
	datasetName    string
	datasetKind    string
	datasetComment *string
	columnName     string
	columnType     string
	columnNullable bool
	columnComment  *string
	ordinal        int
}

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

	discoveryRows := make([]discoveryRow, 0, 64)

	for rows.Next() {
		var row discoveryRow

		if err := rows.Scan(
			&row.schemaName,
			&row.datasetName,
			&row.datasetKind,
			&row.datasetComment,
			&row.columnName,
			&row.columnType,
			&row.columnNullable,
			&row.columnComment,
			&row.ordinal,
		); err != nil {
			return nil, fmt.Errorf("scan postgres schema row: %w", err)
		}
		discoveryRows = append(discoveryRows, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate postgres schema rows: %w", err)
	}

	scannedDatasets, err := buildDiscoveryDatasets(discoveryRows, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	for i := range scannedDatasets {
		profileErr := profileDataset(ctx, pool, &scannedDatasets[i])
		applyProfileStatus(&scannedDatasets[i], profileErr)
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

func buildDiscoveryDatasets(rows []discoveryRow, discoveredAt time.Time) ([]contracts.ScannedDataset, error) {
	datasets := make(map[string]*contracts.ScannedDataset)
	order := make([]string, 0, len(rows))

	for _, row := range rows {
		key := row.schemaName + "." + row.datasetName
		if _, ok := datasets[key]; !ok {
			metadataJSON, err := json.Marshal(map[string]any{
				"schema": row.schemaName,
			})
			if err != nil {
				return nil, fmt.Errorf("marshal dataset metadata: %w", err)
			}

			order = append(order, key)
			datasets[key] = &contracts.ScannedDataset{
				Dataset: model.Dataset{
					Kind:          mapDatasetKind(row.datasetKind),
					DatasetKey:    key,
					Name:          row.datasetName,
					Location:      key,
					Comment:       row.datasetComment,
					DiscoveredAt:  discoveredAt,
					ProfileStatus: types.ProfileStatusDiscoveredOnly,
					MetadataJSON:  metadataJSON,
				},
			}
		}

		dataset := datasets[key]
		dataset.Columns = append(dataset.Columns, contracts.ScannedColumn{
			Column: model.Column{
				Name:            row.columnName,
				OriginalType:    row.columnType,
				NormalizedType:  shared.NormalizeType(row.columnType),
				IsNullable:      row.columnNullable,
				Comment:         row.columnComment,
				OrdinalPosition: row.ordinal,
			},
		})
	}

	result := make([]contracts.ScannedDataset, 0, len(order))
	for _, key := range order {
		result = append(result, *datasets[key])
	}

	return result, nil
}

func applyProfileStatus(dataset *contracts.ScannedDataset, err error) {
	if err != nil {
		errMessage := err.Error()
		dataset.Dataset.ProfileStatus = types.ProfileStatusFailed
		dataset.Dataset.ProfileError = &errMessage
		return
	}

	dataset.Dataset.ProfileStatus = types.ProfileStatusProfiled
	dataset.Dataset.ProfileError = nil
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
