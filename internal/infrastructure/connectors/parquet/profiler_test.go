package parquet

import (
	"testing"
	"time"

	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"
)

func TestFlattenParquetRow(t *testing.T) {
	t.Parallel()

	values := make(map[string]any)
	flattenParquetRow("", map[string]any{
		"id":   int64(1),
		"name": "Alice",
		"profile": map[string]any{
			"city": "Moscow",
		},
	}, values)

	if values["id"] != int64(1) {
		t.Fatalf("expected id to be flattened")
	}
	if values["profile.city"] != "Moscow" {
		t.Fatalf("expected nested field to be flattened, got %#v", values["profile.city"])
	}
}

func TestParquetColumnProfileFinalize(t *testing.T) {
	t.Parallel()

	profile := newParquetColumnProfile(testColumnNumber())
	profile.Observe(int64(10))
	profile.Observe(int64(15))
	profile.Observe(nil)

	stat, topValues, err := profile.Finalize()
	if err != nil {
		t.Fatalf("Finalize returned error: %v", err)
	}
	if stat.NonNullCount != 2 || stat.NullCount != 1 {
		t.Fatalf("unexpected counts: %+v", stat)
	}
	if string(stat.MinValueJSON) != "10" || string(stat.MaxValueJSON) != "15" {
		t.Fatalf("unexpected min/max: %s / %s", stat.MinValueJSON, stat.MaxValueJSON)
	}
	if len(topValues) != 2 {
		t.Fatalf("expected 2 top values, got %d", len(topValues))
	}
}

func TestAsTimestamp(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	if got, ok := asTimestamp(now); !ok || !got.Equal(now) {
		t.Fatalf("expected time.Time to be parsed")
	}

	if _, ok := asTimestamp("2026-03-27T10:20:30Z"); !ok {
		t.Fatalf("expected RFC3339 string to be parsed")
	}
}

func testColumnNumber() model.Column {
	return model.Column{
		Name:           "age",
		OriginalType:   "INT64",
		NormalizedType: types.CanonicalTypeNumber,
	}
}
