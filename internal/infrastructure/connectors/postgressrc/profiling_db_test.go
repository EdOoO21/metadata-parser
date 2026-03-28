package postgressrc

import (
	"context"
	"errors"
	"testing"

	"github.com/EdOoO21/metadata-parser/internal/application/contracts"
	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeProfileRow struct {
	scanFn func(dest ...any) error
}

func (r fakeProfileRow) Scan(dest ...any) error {
	return r.scanFn(dest...)
}

type fakeProfileRows struct {
	index int
	rows  [][]any
	err   error
}

func (r *fakeProfileRows) Close() {}
func (r *fakeProfileRows) Err() error {
	return r.err
}
func (r *fakeProfileRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}
func (r *fakeProfileRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}
func (r *fakeProfileRows) Next() bool {
	if r.index >= len(r.rows) {
		return false
	}
	r.index++
	return true
}
func (r *fakeProfileRows) Scan(dest ...any) error {
	if r.index == 0 || r.index > len(r.rows) {
		return errors.New("scan without row")
	}
	row := r.rows[r.index-1]
	if len(row) < len(dest) {
		return errors.New("short row")
	}
	for i := range dest {
		switch target := dest[i].(type) {
		case *string:
			*target = row[i].(string)
		case *int64:
			*target = row[i].(int64)
		default:
			return errors.New("unsupported scan dest")
		}
	}
	return nil
}
func (r *fakeProfileRows) Values() ([]any, error) {
	return nil, nil
}
func (r *fakeProfileRows) RawValues() [][]byte {
	return nil
}
func (r *fakeProfileRows) Conn() *pgx.Conn {
	return nil
}

type fakeProfileDB struct {
	queryRowFn func(ctx context.Context, sql string, args ...any) pgx.Row
	queryFn    func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (db *fakeProfileDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return db.queryRowFn(ctx, sql, args...)
}

func (db *fakeProfileDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return db.queryFn(ctx, sql, args...)
}

func TestQueryRowCountAndMinMax(t *testing.T) {
	t.Parallel()

	db := &fakeProfileDB{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			switch {
			case len(args) == 0 && sql == `SELECT COUNT(*) FROM "public"."people"`:
				return fakeProfileRow{scanFn: func(dest ...any) error {
					*(dest[0].(*int64)) = 7
					return nil
				}}
			default:
				return fakeProfileRow{scanFn: func(dest ...any) error {
					min := `"2026-03-28T10:00:00Z"`
					max := `"2026-03-29T10:00:00Z"`
					*(dest[0].(**string)) = &min
					*(dest[1].(**string)) = &max
					return nil
				}}
			}
		},
	}

	rowCount, err := queryRowCount(context.Background(), db, "public", "people")
	if err != nil || rowCount != 7 {
		t.Fatalf("unexpected row count result: %d %v", rowCount, err)
	}

	minJSON, maxJSON, err := queryMinMax(context.Background(), db, `"public"."people"`, `"created_at"`)
	if err != nil {
		t.Fatalf("unexpected min/max error: %v", err)
	}
	if string(minJSON) != `"2026-03-28T10:00:00Z"` || string(maxJSON) != `"2026-03-29T10:00:00Z"` {
		t.Fatalf("unexpected min/max: %s %s", string(minJSON), string(maxJSON))
	}
}

func TestQueryTopValuesAndErrors(t *testing.T) {
	t.Parallel()

	db := &fakeProfileDB{
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return &fakeProfileRows{
				rows: [][]any{
					{`"Alice"`, int64(2)},
					{`"Bob"`, int64(1)},
				},
			}, nil
		},
	}

	values, err := queryTopValues(context.Background(), db, `"public"."people"`, `"full_name"`)
	if err != nil {
		t.Fatalf("unexpected top values error: %v", err)
	}
	if len(values) != 2 || values[0].Rank != 1 || string(values[0].ValueJSON) != `"Alice"` {
		t.Fatalf("unexpected top values: %+v", values)
	}

	db = &fakeProfileDB{
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return nil, errors.New("query failed")
		},
	}
	if _, err := queryTopValues(context.Background(), db, `"public"."people"`, `"full_name"`); err == nil {
		t.Fatal("expected query error")
	}
}

func TestProfileColumnAndDatasetWithDB(t *testing.T) {
	t.Parallel()

	db := &fakeProfileDB{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			switch {
			case sql == `SELECT COUNT(*) FROM "public"."people"`:
				return fakeProfileRow{scanFn: func(dest ...any) error {
					*(dest[0].(*int64)) = 2
					return nil
				}}
			case sql == `SELECT COUNT("age"), COUNT(*) - COUNT("age"), COUNT(DISTINCT "age") FROM "public"."people"`:
				return fakeProfileRow{scanFn: func(dest ...any) error {
					*(dest[0].(*int64)) = 2
					*(dest[1].(*int64)) = 0
					*(dest[2].(*int64)) = 2
					return nil
				}}
			default:
				return fakeProfileRow{scanFn: func(dest ...any) error {
					min := "10"
					max := "15"
					*(dest[0].(**string)) = &min
					*(dest[1].(**string)) = &max
					return nil
				}}
			}
		},
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return &fakeProfileRows{
				rows: [][]any{
					{"10", int64(1)},
					{"15", int64(1)},
				},
			}, nil
		},
	}

	stat, topValues, err := profileColumn(context.Background(), db, "public", "people", model.Column{
		Name:           "age",
		NormalizedType: types.CanonicalTypeNumber,
	})
	if err != nil {
		t.Fatalf("unexpected profileColumn error: %v", err)
	}
	if stat.NonNullCount != 2 || string(stat.MinValueJSON) != "10" {
		t.Fatalf("unexpected stat: %+v", stat)
	}
	if len(topValues) != 2 {
		t.Fatalf("unexpected top values: %+v", topValues)
	}

	dataset := &contracts.ScannedDataset{
		Dataset: model.Dataset{Location: "public.people"},
		Columns: []contracts.ScannedColumn{
			{Column: model.Column{Name: "age", NormalizedType: types.CanonicalTypeNumber}},
		},
	}
	if err := profileDatasetWithDB(context.Background(), db, dataset); err != nil {
		t.Fatalf("unexpected profileDatasetWithDB error: %v", err)
	}
	if dataset.Dataset.RowCount == nil || *dataset.Dataset.RowCount != 2 {
		t.Fatalf("unexpected dataset row count: %+v", dataset.Dataset.RowCount)
	}
	if dataset.Columns[0].Stat == nil || len(dataset.Columns[0].TopValues) != 2 {
		t.Fatalf("unexpected dataset profile: %+v", dataset.Columns[0])
	}
}

func TestProfileDatasetWithDB_InvalidLocationAndColumnError(t *testing.T) {
	t.Parallel()

	db := &fakeProfileDB{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return fakeProfileRow{scanFn: func(dest ...any) error {
				if len(dest) == 1 {
					*(dest[0].(*int64)) = 1
					return nil
				}
				return errors.New("stats failed")
			}}
		},
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return &fakeProfileRows{}, nil
		},
	}

	if err := profileDatasetWithDB(context.Background(), db, &contracts.ScannedDataset{
		Dataset: model.Dataset{Location: "broken"},
	}); err == nil {
		t.Fatal("expected invalid location error")
	}

	err := profileDatasetWithDB(context.Background(), db, &contracts.ScannedDataset{
		Dataset: model.Dataset{Location: "public.people"},
		Columns: []contracts.ScannedColumn{
			{Column: model.Column{Name: "age", NormalizedType: types.CanonicalTypeNumber}},
		},
	})
	if err == nil {
		t.Fatal("expected profile column error")
	}
}
