package filescsv

import (
	"encoding/json"
	"testing"

	"github.com/EdOoO21/metadata-parser/internal/domain/types"
)

func TestCSVProfiler_BuildDataset_ProfiledPassport(t *testing.T) {
	t.Parallel()

	profiler := NewCSVProfiler()
	result := &CSVParseResult{
		Headers: []string{"id", "price", "active", "created_at", "city"},
		Rows: []CSVRow{
			{RecordNumber: 1, Values: map[string]string{
				"id": "1", "price": "12.5", "active": "true", "created_at": "2024-01-01T10:00:00Z", "city": "Moscow",
			}},
			{RecordNumber: 2, Values: map[string]string{
				"id": "2", "price": "14.0", "active": "false", "created_at": "2024-01-02T10:00:00Z", "city": "Moscow",
			}},
			{RecordNumber: 3, Values: map[string]string{
				"id": "3", "price": "", "active": "true", "created_at": "2024-01-03T10:00:00Z", "city": "Berlin",
			}},
		},
	}

	dataset, err := profiler.BuildDataset("/tmp/orders.csv", result, 3, 10)
	if err != nil {
		t.Fatalf("BuildDataset returned error: %v", err)
	}

	if dataset.Dataset.ProfileStatus != types.ProfileStatusProfiled {
		t.Fatalf("unexpected profile status: %q", dataset.Dataset.ProfileStatus)
	}
	if dataset.Dataset.RowCount == nil || *dataset.Dataset.RowCount != 3 {
		t.Fatalf("unexpected row count: %#v", dataset.Dataset.RowCount)
	}
	if len(dataset.Columns) != 5 {
		t.Fatalf("unexpected columns len: %d", len(dataset.Columns))
	}

	price := dataset.Columns[1]
	if price.Column.NormalizedType != types.CanonicalTypeNumber {
		t.Fatalf("unexpected price normalized type: %q", price.Column.NormalizedType)
	}
	if !price.Column.IsNullable {
		t.Fatalf("expected price to be nullable")
	}
	if price.Stat == nil || price.Stat.NullCount != 1 || price.Stat.NonNullCount != 2 {
		t.Fatalf("unexpected price stat: %#v", price.Stat)
	}

	active := dataset.Columns[2]
	if active.Column.NormalizedType != types.CanonicalTypeBoolean {
		t.Fatalf("unexpected active normalized type: %q", active.Column.NormalizedType)
	}

	createdAt := dataset.Columns[3]
	if createdAt.Column.NormalizedType != types.CanonicalTypeTimestamp {
		t.Fatalf("unexpected created_at normalized type: %q", createdAt.Column.NormalizedType)
	}
	if createdAt.Stat == nil || len(createdAt.Stat.MinValueJSON) == 0 || len(createdAt.Stat.MaxValueJSON) == 0 {
		t.Fatalf("expected timestamp min/max to be set")
	}

	city := dataset.Columns[4]
	if city.Column.NormalizedType != types.CanonicalTypeString {
		t.Fatalf("unexpected city normalized type: %q", city.Column.NormalizedType)
	}
	if len(city.TopValues) != 2 {
		t.Fatalf("unexpected top values len: %d", len(city.TopValues))
	}

	var topCity string
	if err := json.Unmarshal(city.TopValues[0].ValueJSON, &topCity); err != nil {
		t.Fatalf("unmarshal top value: %v", err)
	}
	if topCity != "Moscow" || city.TopValues[0].OccurrenceCount != 2 {
		t.Fatalf("unexpected top city: %q count=%d", topCity, city.TopValues[0].OccurrenceCount)
	}
}

func TestCSVProfiler_AllNullColumnFallsBackToString(t *testing.T) {
	t.Parallel()

	profiler := NewCSVProfiler()
	result := &CSVParseResult{
		Headers: []string{"notes"},
		Rows: []CSVRow{
			{RecordNumber: 1, Values: map[string]string{"notes": ""}},
			{RecordNumber: 2, Values: map[string]string{"notes": " "}},
		},
	}

	dataset, err := profiler.BuildDataset("/tmp/notes.csv", result, 2, 10)
	if err != nil {
		t.Fatalf("BuildDataset returned error: %v", err)
	}

	column := dataset.Columns[0]
	if column.Column.NormalizedType != types.CanonicalTypeString {
		t.Fatalf("unexpected normalized type: %q", column.Column.NormalizedType)
	}
	if !column.Column.IsNullable {
		t.Fatalf("expected nullable column")
	}
	if column.Stat == nil || column.Stat.NullCount != 2 || column.Stat.NonNullCount != 0 {
		t.Fatalf("unexpected stat: %#v", column.Stat)
	}
}

func TestCSVProfiler_PhoneLikeValuesStayString(t *testing.T) {
	t.Parallel()

	profiler := NewCSVProfiler()
	result := &CSVParseResult{
		Headers: []string{"phone"},
		Rows: []CSVRow{
			{RecordNumber: 1, Values: map[string]string{"phone": "+79990000001"}},
			{RecordNumber: 2, Values: map[string]string{"phone": "+79990000002"}},
			{RecordNumber: 3, Values: map[string]string{"phone": ""}},
		},
	}

	dataset, err := profiler.BuildDataset("/tmp/phones.csv", result, 3, 10)
	if err != nil {
		t.Fatalf("BuildDataset returned error: %v", err)
	}

	column := dataset.Columns[0]
	if column.Column.NormalizedType != types.CanonicalTypeString {
		t.Fatalf("unexpected normalized type: %q", column.Column.NormalizedType)
	}
	if !column.Column.IsNullable {
		t.Fatalf("expected nullable phone column")
	}
	if column.Stat == nil || column.Stat.NonNullCount != 2 || column.Stat.NullCount != 1 {
		t.Fatalf("unexpected stat: %#v", column.Stat)
	}
}
