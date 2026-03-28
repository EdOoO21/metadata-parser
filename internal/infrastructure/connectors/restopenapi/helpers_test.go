package restopenapi

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"
)

func TestBuildColumnsFromSchemaAndHelpers(t *testing.T) {
	t.Parallel()

	components := openAPIComponents{
		Schemas: map[string]*openAPISchema{
			"User": {
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]*openAPISchema{
					"id":    {Type: "integer", Description: "identifier"},
					"email": {Type: "string", Nullable: true},
				},
			},
		},
	}

	schema := &openAPISchema{Ref: "#/components/schemas/User"}
	resolved := resolveResponseSchema(schema, components)
	if resolved == nil || resolved.Type != "object" {
		t.Fatalf("unexpected resolved schema: %+v", resolved)
	}

	columns := buildColumnsFromSchema(schema, components)
	if len(columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(columns))
	}
	if columns[0].Column.Name != "email" || !columns[0].Column.IsNullable {
		t.Fatalf("unexpected first column: %+v", columns[0].Column)
	}
	if columns[1].Column.Name != "id" || columns[1].Column.NormalizedType != types.CanonicalTypeNumber {
		t.Fatalf("unexpected second column: %+v", columns[1].Column)
	}

	if got := inferTypeFromSchema(&openAPISchema{Items: &openAPISchema{Type: "string"}}); got != "array" {
		t.Fatalf("unexpected inferred type: %s", got)
	}
	if got := inferTypeFromSchema(&openAPISchema{Properties: map[string]*openAPISchema{"a": {Type: "string"}}}); got != "object" {
		t.Fatalf("unexpected inferred object type: %s", got)
	}
	if got := joinURL("http://localhost:8080/", "/users"); got != "http://localhost:8080/users" {
		t.Fatalf("unexpected joined url: %s", got)
	}
	if got := firstNonEmpty("", " x ", "y"); got != " x " {
		t.Fatalf("unexpected first non-empty: %q", got)
	}

	values := []string{"b", "a", "c"}
	sortStrings(values)
	if values[0] != "a" || values[2] != "c" {
		t.Fatalf("unexpected sorted strings: %+v", values)
	}
}

func TestResponseRowsAndFlattenRESTValue(t *testing.T) {
	t.Parallel()

	rows := responseRows(map[string]any{"id": 1})
	if len(rows) != 1 {
		t.Fatalf("expected single wrapped row, got %d", len(rows))
	}

	out := map[string]any{}
	flattenRESTValue("", map[string]any{
		"profile": map[string]any{"name": "Ivan"},
		"tags":    []any{"a", "b"},
	}, out)
	if out["profile.name"] != "Ivan" {
		t.Fatalf("unexpected flattened map: %+v", out)
	}
	if _, ok := out["tags"]; !ok {
		t.Fatalf("expected tags in flattened map: %+v", out)
	}
}

func TestRESTColumnProfileFinalizeAndConversions(t *testing.T) {
	t.Parallel()

	numberProfile := newRESTColumnProfile(model.Column{
		Name:           "amount",
		NormalizedType: types.CanonicalTypeNumber,
	})
	numberProfile.Observe(10)
	numberProfile.Observe(json.Number("12.5"))
	numberProfile.Observe(nil)

	stat, topValues, err := numberProfile.Finalize()
	if err != nil {
		t.Fatalf("unexpected finalize error: %v", err)
	}
	if stat.NonNullCount != 2 || stat.NullCount != 1 || string(stat.MinValueJSON) != "10" {
		t.Fatalf("unexpected number stat: %+v", stat)
	}
	if len(topValues) != 2 {
		t.Fatalf("unexpected top values: %+v", topValues)
	}

	tsProfile := newRESTColumnProfile(model.Column{
		Name:           "created_at",
		NormalizedType: types.CanonicalTypeTimestamp,
	})
	tsProfile.Observe("2026-03-28T10:00:00Z")
	tsProfile.Observe("2026-03-29T10:00:00Z")
	stat, _, err = tsProfile.Finalize()
	if err != nil {
		t.Fatalf("unexpected timestamp finalize error: %v", err)
	}
	if stat.MinValueJSON == nil || stat.MaxValueJSON == nil {
		t.Fatalf("expected timestamp bounds: %+v", stat)
	}

	if value, ok := restAsFloat64("12.5"); !ok || value != 12.5 {
		t.Fatalf("unexpected float conversion: %v %v", value, ok)
	}
	if _, ok := restAsFloat64(struct{}{}); ok {
		t.Fatal("expected invalid float conversion")
	}
	if ts, ok := restAsTimestamp("2026-03-28"); !ok || ts.IsZero() {
		t.Fatalf("unexpected timestamp conversion: %v %v", ts, ok)
	}
	if _, ok := restAsTimestamp(123); ok {
		t.Fatal("expected invalid timestamp conversion")
	}
}

func TestRESTColumnProfileObserveInvalidJSONCountsAsNull(t *testing.T) {
	t.Parallel()

	profile := newRESTColumnProfile(model.Column{Name: "payload"})
	profile.Observe(make(chan int))
	stat, _, err := profile.Finalize()
	if err != nil {
		t.Fatalf("unexpected finalize error: %v", err)
	}
	if stat.NullCount != 1 || stat.NonNullCount != 0 {
		t.Fatalf("unexpected invalid-json accounting: %+v", stat)
	}
}

func TestRESTAsTimestampTimeLayouts(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		time.Now().UTC().Format(time.RFC3339),
		"2026-03-28 10:20:30",
		"2026-03-28",
	} {
		if _, ok := restAsTimestamp(value); !ok {
			t.Fatalf("expected layout to parse: %s", value)
		}
	}
}
