package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeRow struct {
	scanFn func(dest ...any) error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.scanFn == nil {
		return nil
	}
	return r.scanFn(dest...)
}

type fakeBatchResults struct {
	execCalls int
	execErrAt int
	closeErr  error
}

func (r *fakeBatchResults) Exec() (pgconn.CommandTag, error) {
	r.execCalls++
	if r.execErrAt > 0 && r.execCalls == r.execErrAt {
		return pgconn.CommandTag{}, errors.New("batch exec failed")
	}
	return pgconn.CommandTag{}, nil
}

func (r *fakeBatchResults) Query() (pgx.Rows, error) {
	return nil, nil
}

func (r *fakeBatchResults) QueryRow() pgx.Row {
	return fakeRow{}
}

func (r *fakeBatchResults) Close() error {
	return r.closeErr
}

type fakeDB struct {
	queryRowFn  func(ctx context.Context, sql string, args ...any) pgx.Row
	queryFn     func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	execFn      func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	sendBatchFn func(ctx context.Context, batch *pgx.Batch) pgx.BatchResults
}

func (db *fakeDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if db.execFn == nil {
		return pgconn.CommandTag{}, nil
	}
	return db.execFn(ctx, sql, args...)
}

func (db *fakeDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if db.queryFn == nil {
		return nil, nil
	}
	return db.queryFn(ctx, sql, args...)
}

func (db *fakeDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if db.queryRowFn == nil {
		return fakeRow{}
	}
	return db.queryRowFn(ctx, sql, args...)
}

func (db *fakeDB) SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults {
	if db.sendBatchFn == nil {
		return &fakeBatchResults{}
	}
	return db.sendBatchFn(ctx, batch)
}

func TestRepositoryEnsureSourceSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	description := "Main source"
	createdAt := time.Date(2026, 3, 17, 10, 0, 0, 0, time.UTC)

	var capturedArgs []any
	repo := newRepositoryWithDB(nil, &fakeDB{
		queryRowFn: func(_ context.Context, _ string, args ...any) pgx.Row {
			capturedArgs = append([]any(nil), args...)
			return fakeRow{scanFn: func(dest ...any) error {
				*(dest[0].(*int64)) = 11
				*(dest[1].(*string)) = "demo_files"
				*(dest[2].(*string)) = string(types.SourceKindFiles)
				*(dest[3].(**string)) = &description
				*(dest[4].(*time.Time)) = createdAt
				return nil
			}}
		},
	})

	got, err := repo.EnsureSource(ctx, model.Source{
		Name:        "demo_files",
		Kind:        types.SourceKindFiles,
		Description: &description,
	})
	if err != nil {
		t.Fatalf("EnsureSource returned error: %v", err)
	}

	if got.ID != 11 || got.Name != "demo_files" || got.Kind != types.SourceKindFiles {
		t.Fatalf("unexpected source: %+v", got)
	}
	if got.Description == nil || *got.Description != description {
		t.Fatalf("unexpected description: %+v", got.Description)
	}
	if !got.CreatedAt.Equal(createdAt) {
		t.Fatalf("unexpected created_at: %v", got.CreatedAt)
	}

	expectedArgs := []any{"demo_files", string(types.SourceKindFiles), description}
	if !reflect.DeepEqual(capturedArgs, expectedArgs) {
		t.Fatalf("unexpected query args:\n got: %#v\nwant: %#v", capturedArgs, expectedArgs)
	}
}

func TestRepositoryEnsureSourceScanError(t *testing.T) {
	t.Parallel()

	repo := newRepositoryWithDB(nil, &fakeDB{
		queryRowFn: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return fakeRow{scanFn: func(dest ...any) error {
				return errors.New("scan failed")
			}}
		},
	})

	_, err := repo.EnsureSource(context.Background(), model.Source{Name: "broken", Kind: types.SourceKindFiles})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "ensure source broken: scan failed" {
		t.Fatalf("unexpected error: %s", got)
	}
}

func TestRepositoryCreateRunSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	startedAt := time.Date(2026, 3, 17, 11, 0, 0, 0, time.UTC)
	finishedAt := time.Date(2026, 3, 17, 11, 5, 0, 0, time.UTC)
	payload := []byte(`{"version":1,"sources":3}`)
	errMsg := "partial"

	var capturedArgs []any
	repo := newRepositoryWithDB(nil, &fakeDB{
		queryRowFn: func(_ context.Context, _ string, args ...any) pgx.Row {
			capturedArgs = append([]any(nil), args...)
			return fakeRow{scanFn: func(dest ...any) error {
				*(dest[0].(*int64)) = 101
				*(dest[1].(*time.Time)) = startedAt
				*(dest[2].(**time.Time)) = &finishedAt
				*(dest[3].(*string)) = string(types.RunStatusPartial)
				*(dest[4].(*string)) = "cfg-hash"
				b := append([]byte(nil), payload...)
				*(dest[5].(*[]byte)) = b
				*(dest[6].(**string)) = &errMsg
				return nil
			}}
		},
	})

	got, err := repo.CreateRun(ctx, model.Run{
		StartedAt:          startedAt,
		FinishedAt:         &finishedAt,
		Status:             types.RunStatusPartial,
		ConfigHash:         "cfg-hash",
		ConfigSnapshotJSON: payload,
		ErrorMessage:       &errMsg,
	})
	if err != nil {
		t.Fatalf("CreateRun returned error: %v", err)
	}

	if got.ID != 101 || got.Status != types.RunStatusPartial || got.ConfigHash != "cfg-hash" {
		t.Fatalf("unexpected run: %+v", got)
	}
	if got.FinishedAt == nil || !got.FinishedAt.Equal(finishedAt) {
		t.Fatalf("unexpected finishedAt: %+v", got.FinishedAt)
	}
	if got.ErrorMessage == nil || *got.ErrorMessage != errMsg {
		t.Fatalf("unexpected error message: %+v", got.ErrorMessage)
	}
	if string(got.ConfigSnapshotJSON) != string(payload) {
		t.Fatalf("unexpected payload: %s", string(got.ConfigSnapshotJSON))
	}
	payload[0] = 'X'
	if string(got.ConfigSnapshotJSON) == string(payload) {
		t.Fatal("expected returned JSON to be cloned")
	}

	if len(capturedArgs) != 6 {
		t.Fatalf("unexpected arg count: %d", len(capturedArgs))
	}
	if capturedArgs[0] != startedAt {
		t.Fatalf("unexpected startedAt arg: %#v", capturedArgs[0])
	}
	if capturedArgs[2] != string(types.RunStatusPartial) {
		t.Fatalf("unexpected status arg: %#v", capturedArgs[2])
	}
}

func TestRepositoryUpdateRunStatusExecArgs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	finishedAt := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)
	errMsg := "failed to collect"

	var capturedArgs []any
	repo := newRepositoryWithDB(nil, &fakeDB{
		execFn: func(_ context.Context, _ string, args ...any) (pgconn.CommandTag, error) {
			capturedArgs = append([]any(nil), args...)
			return pgconn.CommandTag{}, nil
		},
	})

	err := repo.UpdateRunStatus(ctx, 55, types.RunStatusFailed, &finishedAt, &errMsg)
	if err != nil {
		t.Fatalf("UpdateRunStatus returned error: %v", err)
	}

	expectedArgs := []any{int64(55), string(types.RunStatusFailed), finishedAt, errMsg}
	if !reflect.DeepEqual(capturedArgs, expectedArgs) {
		t.Fatalf("unexpected exec args:\n got: %#v\nwant: %#v", capturedArgs, expectedArgs)
	}
}

func TestRepositoryCreateRunSourceSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	startedAt := time.Date(2026, 3, 17, 13, 0, 0, 0, time.UTC)
	payload := []byte(`{"path":"./demo/files"}`)

	repo := newRepositoryWithDB(nil, &fakeDB{
		queryRowFn: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return fakeRow{scanFn: func(dest ...any) error {
				*(dest[0].(*int64)) = 201
				*(dest[1].(*int64)) = 10
				*(dest[2].(*int64)) = 20
				*(dest[3].(*time.Time)) = startedAt
				*(dest[4].(**time.Time)) = nil
				*(dest[5].(*string)) = string(types.RunStatusRunning)
				*(dest[6].(**string)) = nil
				b := append([]byte(nil), payload...)
				*(dest[7].(*[]byte)) = b
				return nil
			}}
		},
	})

	got, err := repo.CreateRunSource(ctx, model.RunSource{
		RunID:               10,
		SourceID:            20,
		StartedAt:           startedAt,
		Status:              types.RunStatusRunning,
		EffectiveConfigJSON: payload,
	})
	if err != nil {
		t.Fatalf("CreateRunSource returned error: %v", err)
	}

	if got.ID != 201 || got.RunID != 10 || got.SourceID != 20 || got.Status != types.RunStatusRunning {
		t.Fatalf("unexpected run source: %+v", got)
	}
	if string(got.EffectiveConfigJSON) != string(payload) {
		t.Fatalf("unexpected payload: %s", string(got.EffectiveConfigJSON))
	}
}

func TestRepositoryUpdateRunSourceStatusExecArgs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	finishedAt := time.Date(2026, 3, 17, 14, 0, 0, 0, time.UTC)

	var capturedArgs []any
	repo := newRepositoryWithDB(nil, &fakeDB{
		execFn: func(_ context.Context, _ string, args ...any) (pgconn.CommandTag, error) {
			capturedArgs = append([]any(nil), args...)
			return pgconn.CommandTag{}, nil
		},
	})

	err := repo.UpdateRunSourceStatus(ctx, 77, types.RunStatusSuccess, &finishedAt, nil)
	if err != nil {
		t.Fatalf("UpdateRunSourceStatus returned error: %v", err)
	}

	expectedArgs := []any{int64(77), string(types.RunStatusSuccess), finishedAt, nil}
	if !reflect.DeepEqual(capturedArgs, expectedArgs) {
		t.Fatalf("unexpected exec args:\n got: %#v\nwant: %#v", capturedArgs, expectedArgs)
	}
}

func TestRepositoryCreateDatasetSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	discoveredAt := time.Date(2026, 3, 17, 15, 0, 0, 0, time.UTC)
	comment := "customers export"
	rowCount := int64(125)
	profileError := "missing params"
	metadata := []byte(`{"method":"GET","path":"/customers"}`)

	repo := newRepositoryWithDB(nil, &fakeDB{
		queryRowFn: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return fakeRow{scanFn: func(dest ...any) error {
				*(dest[0].(*int64)) = 301
				*(dest[1].(*int64)) = 22
				*(dest[2].(*string)) = string(types.DatasetKindEndpoint)
				*(dest[3].(*string)) = "GET /customers"
				*(dest[4].(*string)) = "Customers endpoint"
				*(dest[5].(*string)) = "/customers"
				*(dest[6].(**string)) = &comment
				*(dest[7].(**int64)) = &rowCount
				*(dest[8].(*time.Time)) = discoveredAt
				*(dest[9].(*string)) = string(types.ProfileStatusSkippedRequiresParams)
				*(dest[10].(**string)) = &profileError
				b := append([]byte(nil), metadata...)
				*(dest[11].(*[]byte)) = b
				return nil
			}}
		},
	})

	got, err := repo.CreateDataset(ctx, model.Dataset{
		RunSourceID:   22,
		Kind:          types.DatasetKindEndpoint,
		DatasetKey:    "GET /customers",
		Name:          "Customers endpoint",
		Location:      "/customers",
		Comment:       &comment,
		RowCount:      &rowCount,
		DiscoveredAt:  discoveredAt,
		ProfileStatus: types.ProfileStatusSkippedRequiresParams,
		ProfileError:  &profileError,
		MetadataJSON:  metadata,
	})
	if err != nil {
		t.Fatalf("CreateDataset returned error: %v", err)
	}

	if got.ID != 301 || got.Kind != types.DatasetKindEndpoint || got.ProfileStatus != types.ProfileStatusSkippedRequiresParams {
		t.Fatalf("unexpected dataset: %+v", got)
	}
	if got.RowCount == nil || *got.RowCount != rowCount {
		t.Fatalf("unexpected row count: %+v", got.RowCount)
	}
	if got.Comment == nil || *got.Comment != comment {
		t.Fatalf("unexpected comment: %+v", got.Comment)
	}
	if got.ProfileError == nil || *got.ProfileError != profileError {
		t.Fatalf("unexpected profile error: %+v", got.ProfileError)
	}
	if string(got.MetadataJSON) != string(metadata) {
		t.Fatalf("unexpected metadata: %s", string(got.MetadataJSON))
	}
}

func TestRepositoryCreateColumnSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	comment := "primary key"

	repo := newRepositoryWithDB(nil, &fakeDB{
		queryRowFn: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return fakeRow{scanFn: func(dest ...any) error {
				*(dest[0].(*int64)) = 401
				*(dest[1].(*int64)) = 301
				*(dest[2].(*string)) = "id"
				*(dest[3].(*string)) = "integer"
				*(dest[4].(*string)) = string(types.CanonicalTypeNumber)
				*(dest[5].(*bool)) = false
				*(dest[6].(**string)) = &comment
				*(dest[7].(*int)) = 1
				return nil
			}}
		},
	})

	got, err := repo.CreateColumn(ctx, model.Column{
		DatasetID:       301,
		Name:            "id",
		OriginalType:    "integer",
		NormalizedType:  types.CanonicalTypeNumber,
		IsNullable:      false,
		Comment:         &comment,
		OrdinalPosition: 1,
	})
	if err != nil {
		t.Fatalf("CreateColumn returned error: %v", err)
	}

	if got.ID != 401 || got.NormalizedType != types.CanonicalTypeNumber || got.OrdinalPosition != 1 {
		t.Fatalf("unexpected column: %+v", got)
	}
	if got.Comment == nil || *got.Comment != comment {
		t.Fatalf("unexpected comment: %+v", got.Comment)
	}
}

func TestRepositoryCreateColumnStatSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	minValue := []byte(`1`)
	maxValue := []byte(`999`)

	repo := newRepositoryWithDB(nil, &fakeDB{
		queryRowFn: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return fakeRow{scanFn: func(dest ...any) error {
				*(dest[0].(*int64)) = 501
				*(dest[1].(*int64)) = 401
				*(dest[2].(*int64)) = 100
				*(dest[3].(*int64)) = 0
				*(dest[4].(*int64)) = 100
				*(dest[5].(*[]byte)) = append([]byte(nil), minValue...)
				*(dest[6].(*[]byte)) = append([]byte(nil), maxValue...)
				return nil
			}}
		},
	})

	got, err := repo.CreateColumnStat(ctx, model.ColumnStat{
		ColumnID:      401,
		NonNullCount:  100,
		NullCount:     0,
		DistinctCount: 100,
		MinValueJSON:  minValue,
		MaxValueJSON:  maxValue,
	})
	if err != nil {
		t.Fatalf("CreateColumnStat returned error: %v", err)
	}

	if got.ID != 501 || got.ColumnID != 401 || got.DistinctCount != 100 {
		t.Fatalf("unexpected stat: %+v", got)
	}
	if string(got.MinValueJSON) != string(minValue) || string(got.MaxValueJSON) != string(maxValue) {
		t.Fatalf("unexpected min/max: min=%s max=%s", got.MinValueJSON, got.MaxValueJSON)
	}
}

func TestRepositoryCreateColumnTopValuesEmptyInput(t *testing.T) {
	t.Parallel()

	called := false
	repo := newRepositoryWithDB(nil, &fakeDB{
		sendBatchFn: func(_ context.Context, _ *pgx.Batch) pgx.BatchResults {
			called = true
			return &fakeBatchResults{}
		},
	})

	if err := repo.CreateColumnTopValues(context.Background(), nil); err != nil {
		t.Fatalf("CreateColumnTopValues returned error: %v", err)
	}
	if called {
		t.Fatal("expected batch sender not to be called for empty input")
	}
}

func TestRepositoryCreateColumnTopValuesSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	results := &fakeBatchResults{}
	called := false

	repo := newRepositoryWithDB(nil, &fakeDB{
		sendBatchFn: func(_ context.Context, batch *pgx.Batch) pgx.BatchResults {
			called = true
			if batch.Len() != 2 {
				t.Fatalf("unexpected batch len: %d", batch.Len())
			}
			return results
		},
	})

	err := repo.CreateColumnTopValues(ctx, []model.ColumnTopValue{
		{ColumnStatID: 501, Rank: 1, ValueJSON: []byte(`"Alice"`), OccurrenceCount: 10},
		{ColumnStatID: 501, Rank: 2, ValueJSON: []byte(`"Bob"`), OccurrenceCount: 5},
	})
	if err != nil {
		t.Fatalf("CreateColumnTopValues returned error: %v", err)
	}
	if !called {
		t.Fatal("expected batch sender to be called")
	}
	if results.execCalls != 2 {
		t.Fatalf("unexpected exec call count: %d", results.execCalls)
	}
}

func TestNullableJSONValidAndInvalid(t *testing.T) {
	t.Parallel()

	valid := nullableJSON([]byte(`{"key":"value"}`))
	validRaw, ok := valid.(json.RawMessage)
	if !ok {
		t.Fatalf("expected valid JSON to stay raw JSON, got %T", valid)
	}
	if string(validRaw) != `{"key":"value"}` {
		t.Fatalf("unexpected raw JSON: %s", string(validRaw))
	}

	invalid := nullableJSON([]byte(`not-json`))
	invalidRaw, ok := invalid.(json.RawMessage)
	if !ok {
		t.Fatalf("expected invalid JSON bytes to stay raw JSON, got %T", invalid)
	}
	if string(invalidRaw) != "not-json" {
		t.Fatalf("unexpected raw JSON fallback: %q", string(invalidRaw))
	}

	if got := nullableJSON(nil); got != nil {
		t.Fatalf("expected nil JSON to stay nil, got %#v", got)
	}
}

func TestCloneBytesReturnsIndependentCopy(t *testing.T) {
	t.Parallel()

	src := []byte(`{"x":1}`)
	cloned := cloneBytes(src)
	if string(cloned) != string(src) {
		t.Fatalf("unexpected clone: %s", string(cloned))
	}

	src[0] = 'X'
	if string(cloned) == string(src) {
		t.Fatal("expected clone to be independent from original slice")
	}

	if cloneBytes(nil) != nil {
		t.Fatal("expected nil input to produce nil output")
	}
}
