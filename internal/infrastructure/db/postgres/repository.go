package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type dbtx interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, args string, arguments ...any) pgx.Row
}

type Repository struct {
	pool *pgxpool.Pool
	db   dbtx
}

var _ appports.CatalogRepository = (*Repository)(nil)

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool: pool,
		db:   pool,
	}
}

func newRepositoryWithDB(pool *pgxpool.Pool, db dbtx) *Repository {
	return &Repository{
		pool: pool,
		db:   db,
	}
}

func (r *Repository) WithTx(ctx context.Context, fn func(repo appports.CatalogRepository) error) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	txRepo := newRepositoryWithDB(r.pool, tx)

	if err := fn(txRepo); err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			return fmt.Errorf("rollback tx after error %v: %w", err, rollbackErr)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (r *Repository) EnsureSource(ctx context.Context, source model.Source) (*model.Source, error) {
	const query = `
INSERT INTO sources (name, kind, description)
VALUES ($1, $2, $3)
ON CONFLICT (name) DO UPDATE
SET kind = EXCLUDED.kind,
    description = EXCLUDED.description
RETURNING id, name, kind, description, created_at
`

	row := r.db.QueryRow(
		ctx,
		query,
		source.Name,
		string(source.Kind),
		nullableString(source.Description),
	)

	var result model.Source
	var kind string
	var description *string

	if err := row.Scan(
		&result.ID,
		&result.Name,
		&kind,
		&description,
		&result.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("ensure source %s: %w", source.Name, err)
	}

	result.Kind = types.SourceKind(kind)
	result.Description = description

	return &result, nil
}

func (r *Repository) CreateRun(ctx context.Context, run model.Run) (*model.Run, error) {
	const query = `
INSERT INTO runs (started_at, finished_at, status, config_hash, config_snapshot_json, error_message)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, started_at, finished_at, status, config_hash, config_snapshot_json, error_message
`

	row := r.db.QueryRow(
		ctx,
		query,
		run.StartedAt,
		nullableTime(run.FinishedAt),
		string(run.Status),
		run.ConfigHash,
		nullableJSON(run.ConfigSnapshotJSON),
		nullableString(run.ErrorMessage),
	)

	return scanRun(row, "create run")
}

func (r *Repository) GetRun(ctx context.Context, runID int64) (*model.Run, error) {
	const query = `
SELECT id, started_at, finished_at, status, config_hash, config_snapshot_json, error_message
FROM runs
WHERE id = $1
`

	row := r.db.QueryRow(ctx, query, runID)
	return scanRun(row, fmt.Sprintf("get run %d", runID))
}

func (r *Repository) ListRecentRuns(ctx context.Context, limit int) ([]model.Run, error) {
	const query = `
SELECT id, started_at, finished_at, status, config_hash, config_snapshot_json, error_message
FROM runs
ORDER BY id DESC
LIMIT $1
`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent runs: %w", err)
	}
	defer rows.Close()

	runs := make([]model.Run, 0, limit)
	for rows.Next() {
		run, err := scanRun(rows, "list recent runs")
		if err != nil {
			return nil, err
		}
		runs = append(runs, *run)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list recent runs: %w", err)
	}

	return runs, nil
}

func (r *Repository) UpdateRunStatus(
	ctx context.Context,
	runID int64,
	status types.RunStatus,
	finishedAt *time.Time,
	errorMessage *string,
) error {
	const query = `
UPDATE runs
SET status = $2,
    finished_at = $3,
    error_message = $4
WHERE id = $1
`

	if _, err := r.db.Exec(
		ctx,
		query,
		runID,
		string(status),
		nullableTime(finishedAt),
		nullableString(errorMessage),
	); err != nil {
		return fmt.Errorf("update run status %d: %w", runID, err)
	}

	return nil
}

func (r *Repository) CreateRunSource(ctx context.Context, runSource model.RunSource) (*model.RunSource, error) {
	const query = `
INSERT INTO run_sources (run_id, source_id, started_at, finished_at, status, error_message, effective_config_json)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, run_id, source_id, started_at, finished_at, status, error_message, effective_config_json
`

	row := r.db.QueryRow(
		ctx,
		query,
		runSource.RunID,
		runSource.SourceID,
		runSource.StartedAt,
		nullableTime(runSource.FinishedAt),
		string(runSource.Status),
		nullableString(runSource.ErrorMessage),
		nullableJSON(runSource.EffectiveConfigJSON),
	)

	return scanRunSource(row, "create run source")
}

func (r *Repository) UpdateRunSourceStatus(
	ctx context.Context,
	runSourceID int64,
	status types.RunStatus,
	finishedAt *time.Time,
	errorMessage *string,
) error {
	const query = `
UPDATE run_sources
SET status = $2,
    finished_at = $3,
    error_message = $4
WHERE id = $1
`

	if _, err := r.db.Exec(
		ctx,
		query,
		runSourceID,
		string(status),
		nullableTime(finishedAt),
		nullableString(errorMessage),
	); err != nil {
		return fmt.Errorf("update run source status %d: %w", runSourceID, err)
	}

	return nil
}

func (r *Repository) CreateDataset(ctx context.Context, dataset model.Dataset) (*model.Dataset, error) {
	const query = `
INSERT INTO datasets (
    run_source_id,
    kind,
    dataset_key,
    name,
    location,
    comment,
    row_count,
    discovered_at,
    profile_status,
    profile_error,
    metadata_json
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id, run_source_id, kind, dataset_key, name, location, comment, row_count, discovered_at, profile_status, profile_error, metadata_json
`

	row := r.db.QueryRow(
		ctx,
		query,
		dataset.RunSourceID,
		string(dataset.Kind),
		dataset.DatasetKey,
		dataset.Name,
		dataset.Location,
		nullableString(dataset.Comment),
		nullableInt64(dataset.RowCount),
		dataset.DiscoveredAt,
		string(dataset.ProfileStatus),
		nullableString(dataset.ProfileError),
		nullableJSON(dataset.MetadataJSON),
	)

	return scanDataset(row, "create dataset")
}

func (r *Repository) CreateColumn(ctx context.Context, column model.Column) (*model.Column, error) {
	const query = `
INSERT INTO columns (
    dataset_id,
    name,
    original_type,
    normalized_type,
    is_nullable,
    comment,
    ordinal_position
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, dataset_id, name, original_type, normalized_type, is_nullable, comment, ordinal_position
`

	row := r.db.QueryRow(
		ctx,
		query,
		column.DatasetID,
		column.Name,
		column.OriginalType,
		string(column.NormalizedType),
		column.IsNullable,
		nullableString(column.Comment),
		column.OrdinalPosition,
	)

	return scanColumn(row, "create column")
}

func (r *Repository) CreateColumnStat(ctx context.Context, stat model.ColumnStat) (*model.ColumnStat, error) {
	const query = `
INSERT INTO column_stats (
    column_id,
    non_null_count,
    null_count,
    distinct_count,
    min_value_json,
    max_value_json
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, column_id, non_null_count, null_count, distinct_count, min_value_json, max_value_json
`

	row := r.db.QueryRow(
		ctx,
		query,
		stat.ColumnID,
		stat.NonNullCount,
		stat.NullCount,
		stat.DistinctCount,
		nullableJSON(stat.MinValueJSON),
		nullableJSON(stat.MaxValueJSON),
	)

	return scanColumnStat(row, "create column stat")
}

func (r *Repository) CreateColumnTopValues(ctx context.Context, values []model.ColumnTopValue) error {
	if len(values) == 0 {
		return nil
	}

	const query = `
INSERT INTO column_top_values (
    column_stat_id,
    rank,
    value_json,
    occurrence_count
)
VALUES ($1, $2, $3, $4)
`

	batch := &pgx.Batch{}
	for _, value := range values {
		batch.Queue(
			query,
			value.ColumnStatID,
			value.Rank,
			marshalJSONValue(value.ValueJSON),
			value.OccurrenceCount,
		)
	}

	results := r.batchSender().SendBatch(ctx, batch)

	for i := range values {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("insert column top value #%d: %w", i, err)
		}
	}

	if err := results.Close(); err != nil {
		return fmt.Errorf("close column top values batch: %w", err)
	}

	return nil
}

func (r *Repository) ListReportRows(ctx context.Context, runID int64) ([]appports.ReportRow, error) {
	const query = `
SELECT
    s.name,
    s.kind,
    d.name,
    d.kind,
    d.dataset_key,
    d.location,
    d.comment,
    d.row_count,
    d.profile_status,
    c.name,
    c.original_type,
    c.normalized_type,
    c.is_nullable,
    c.comment,
    c.ordinal_position
FROM run_sources rs
JOIN sources s ON s.id = rs.source_id
JOIN datasets d ON d.run_source_id = rs.id
JOIN columns c ON c.dataset_id = d.id
WHERE rs.run_id = $1
ORDER BY s.name, d.name, c.ordinal_position, c.name
`

	rows, err := r.db.Query(ctx, query, runID)
	if err != nil {
		return nil, fmt.Errorf("list report rows for run %d: %w", runID, err)
	}
	defer rows.Close()

	result := make([]appports.ReportRow, 0, 64)
	for rows.Next() {
		var item appports.ReportRow
		var sourceKind string
		var datasetKind string
		var datasetComment *string
		var datasetRowCount *int64
		var profileStatus string
		var normalizedType string
		var columnComment *string

		if err := rows.Scan(
			&item.SourceName,
			&sourceKind,
			&item.DatasetName,
			&datasetKind,
			&item.DatasetKey,
			&item.DatasetLocation,
			&datasetComment,
			&datasetRowCount,
			&profileStatus,
			&item.ColumnName,
			&item.ColumnOriginalType,
			&normalizedType,
			&item.ColumnIsNullable,
			&columnComment,
			&item.ColumnOrdinal,
		); err != nil {
			return nil, fmt.Errorf("scan report row: %w", err)
		}

		item.SourceKind = types.SourceKind(sourceKind)
		item.DatasetKind = types.DatasetKind(datasetKind)
		item.DatasetComment = datasetComment
		item.DatasetRowCount = datasetRowCount
		item.DatasetProfileStatus = types.ProfileStatus(profileStatus)
		item.ColumnNormalizedType = types.CanonicalType(normalizedType)
		item.ColumnComment = columnComment

		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate report rows: %w", err)
	}

	return result, nil
}

func (r *Repository) batchSender() interface {
	SendBatch(context.Context, *pgx.Batch) pgx.BatchResults
} {
	if sender, ok := r.db.(interface {
		SendBatch(context.Context, *pgx.Batch) pgx.BatchResults
	}); ok {
		return sender
	}

	return r.pool
}

func scanRun(row pgx.Row, op string) (*model.Run, error) {
	var result model.Run
	var status string
	var configSnapshot []byte
	var errorMessage *string

	if err := row.Scan(
		&result.ID,
		&result.StartedAt,
		&result.FinishedAt,
		&status,
		&result.ConfigHash,
		&configSnapshot,
		&errorMessage,
	); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	result.Status = types.RunStatus(status)
	result.ConfigSnapshotJSON = cloneBytes(configSnapshot)
	result.ErrorMessage = errorMessage

	return &result, nil
}

func scanRunSource(row pgx.Row, op string) (*model.RunSource, error) {
	var result model.RunSource
	var status string
	var effectiveConfig []byte
	var errorMessage *string

	if err := row.Scan(
		&result.ID,
		&result.RunID,
		&result.SourceID,
		&result.StartedAt,
		&result.FinishedAt,
		&status,
		&errorMessage,
		&effectiveConfig,
	); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	result.Status = types.RunStatus(status)
	result.ErrorMessage = errorMessage
	result.EffectiveConfigJSON = cloneBytes(effectiveConfig)

	return &result, nil
}

func scanDataset(row pgx.Row, op string) (*model.Dataset, error) {
	var result model.Dataset
	var kind string
	var profileStatus string
	var comment *string
	var rowCount *int64
	var profileError *string
	var metadata []byte

	if err := row.Scan(
		&result.ID,
		&result.RunSourceID,
		&kind,
		&result.DatasetKey,
		&result.Name,
		&result.Location,
		&comment,
		&rowCount,
		&result.DiscoveredAt,
		&profileStatus,
		&profileError,
		&metadata,
	); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	result.Kind = types.DatasetKind(kind)
	result.Comment = comment
	result.RowCount = rowCount
	result.ProfileStatus = types.ProfileStatus(profileStatus)
	result.ProfileError = profileError
	result.MetadataJSON = cloneBytes(metadata)

	return &result, nil
}

func scanColumn(row pgx.Row, op string) (*model.Column, error) {
	var result model.Column
	var normalizedType string
	var comment *string

	if err := row.Scan(
		&result.ID,
		&result.DatasetID,
		&result.Name,
		&result.OriginalType,
		&normalizedType,
		&result.IsNullable,
		&comment,
		&result.OrdinalPosition,
	); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	result.NormalizedType = types.CanonicalType(normalizedType)
	result.Comment = comment

	return &result, nil
}

func scanColumnStat(row pgx.Row, op string) (*model.ColumnStat, error) {
	var result model.ColumnStat
	var minValue []byte
	var maxValue []byte

	if err := row.Scan(
		&result.ID,
		&result.ColumnID,
		&result.NonNullCount,
		&result.NullCount,
		&result.DistinctCount,
		&minValue,
		&maxValue,
	); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	result.MinValueJSON = cloneBytes(minValue)
	result.MaxValueJSON = cloneBytes(maxValue)

	return &result, nil
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableJSON(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return marshalJSONValue(value)
}

func marshalJSONValue(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return json.RawMessage(cloneBytes(value))
}

func cloneBytes(src []byte) []byte {
	if len(src) == 0 {
		return nil
	}

	dst := make([]byte, len(src))
	copy(dst, src)

	return dst
}
