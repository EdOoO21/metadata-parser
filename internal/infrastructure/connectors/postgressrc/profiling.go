package postgressrc

import (
	"context"
	"fmt"
	"strings"

	"github.com/EdOoO21/metadata-parser/internal/application/contracts"
	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultPostgresTopValuesLimit = 5

func profileDataset(ctx context.Context, pool *pgxpool.Pool, dataset *contracts.ScannedDataset) error {
	schemaName, datasetName, err := parseDatasetLocation(dataset.Dataset.Location)
	if err != nil {
		return err
	}

	rowCount, err := queryRowCount(ctx, pool, schemaName, datasetName)
	if err != nil {
		return err
	}
	dataset.Dataset.RowCount = &rowCount

	for i := range dataset.Columns {
		stat, topValues, err := profileColumn(ctx, pool, schemaName, datasetName, dataset.Columns[i].Column)
		if err != nil {
			return fmt.Errorf("profile column %q: %w", dataset.Columns[i].Column.Name, err)
		}

		dataset.Columns[i].Stat = stat
		dataset.Columns[i].TopValues = topValues
	}

	return nil
}

func profileColumn(
	ctx context.Context,
	pool *pgxpool.Pool,
	schemaName string,
	datasetName string,
	column model.Column,
) (*model.ColumnStat, []model.ColumnTopValue, error) {
	qualifiedName := qualifyName(schemaName, datasetName)
	columnName := quoteIdent(column.Name)

	statsQuery := fmt.Sprintf(
		`SELECT COUNT(%s), COUNT(*) - COUNT(%s), COUNT(DISTINCT %s) FROM %s`,
		columnName,
		columnName,
		columnName,
		qualifiedName,
	)

	stat := &model.ColumnStat{}
	if err := pool.QueryRow(ctx, statsQuery).Scan(&stat.NonNullCount, &stat.NullCount, &stat.DistinctCount); err != nil {
		return nil, nil, fmt.Errorf("query column stats: %w", err)
	}

	if column.NormalizedType == types.CanonicalTypeNumber || column.NormalizedType == types.CanonicalTypeTimestamp {
		minValueJSON, maxValueJSON, err := queryMinMax(ctx, pool, qualifiedName, columnName)
		if err != nil {
			return nil, nil, err
		}
		stat.MinValueJSON = minValueJSON
		stat.MaxValueJSON = maxValueJSON
	}

	topValues, err := queryTopValues(ctx, pool, qualifiedName, columnName)
	if err != nil {
		return nil, nil, err
	}

	return stat, topValues, nil
}

func queryRowCount(ctx context.Context, pool *pgxpool.Pool, schemaName string, datasetName string) (int64, error) {
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, qualifyName(schemaName, datasetName))

	var rowCount int64
	if err := pool.QueryRow(ctx, query).Scan(&rowCount); err != nil {
		return 0, fmt.Errorf("query row count: %w", err)
	}

	return rowCount, nil
}

func queryMinMax(ctx context.Context, pool *pgxpool.Pool, qualifiedName string, columnName string) ([]byte, []byte, error) {
	query := fmt.Sprintf(
		`SELECT to_jsonb(MIN(%s))::text, to_jsonb(MAX(%s))::text FROM %s WHERE %s IS NOT NULL`,
		columnName,
		columnName,
		qualifiedName,
		columnName,
	)

	var minValue *string
	var maxValue *string
	if err := pool.QueryRow(ctx, query).Scan(&minValue, &maxValue); err != nil {
		return nil, nil, fmt.Errorf("query min/max: %w", err)
	}

	return nullableJSONString(minValue), nullableJSONString(maxValue), nil
}

func queryTopValues(ctx context.Context, pool *pgxpool.Pool, qualifiedName string, columnName string) ([]model.ColumnTopValue, error) {
	query := fmt.Sprintf(
		`SELECT to_jsonb(%[1]s)::text, COUNT(*)
FROM %[2]s
WHERE %[1]s IS NOT NULL
GROUP BY %[1]s
ORDER BY COUNT(*) DESC, to_jsonb(%[1]s)::text ASC
LIMIT %[3]d`,
		columnName,
		qualifiedName,
		defaultPostgresTopValuesLimit,
	)

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query top values: %w", err)
	}
	defer rows.Close()

	topValues := make([]model.ColumnTopValue, 0, defaultPostgresTopValuesLimit)
	rank := 1
	for rows.Next() {
		var valueJSON string
		var occurrenceCount int64

		if err := rows.Scan(&valueJSON, &occurrenceCount); err != nil {
			return nil, fmt.Errorf("scan top values row: %w", err)
		}

		topValues = append(topValues, model.ColumnTopValue{
			Rank:            rank,
			ValueJSON:       []byte(valueJSON),
			OccurrenceCount: occurrenceCount,
		})
		rank++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate top values rows: %w", err)
	}

	return topValues, nil
}

func parseDatasetLocation(location string) (string, string, error) {
	parts := strings.SplitN(location, ".", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("invalid dataset location %q", location)
	}

	return parts[0], parts[1], nil
}

func qualifyName(schemaName string, datasetName string) string {
	return quoteIdent(schemaName) + "." + quoteIdent(datasetName)
}

func quoteIdent(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func nullableJSONString(value *string) []byte {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" || strings.EqualFold(trimmed, "null") {
		return nil
	}

	return []byte(trimmed)
}
