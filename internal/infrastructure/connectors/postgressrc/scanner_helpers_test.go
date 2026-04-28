package postgressrc

import (
	"testing"

	"github.com/EdOoO21/metadata-parser/internal/domain/types"
)

func TestMapDatasetKind(t *testing.T) {
	t.Parallel()

	if got := mapDatasetKind("v"); got != types.DatasetKindView {
		t.Fatalf("expected view for relkind v, got %s", got)
	}
	if got := mapDatasetKind("r"); got != types.DatasetKindTable {
		t.Fatalf("expected table for relkind r, got %s", got)
	}
	if got := mapDatasetKind("VIEW"); got != types.DatasetKindView {
		t.Fatalf("expected view, got %s", got)
	}
	if got := mapDatasetKind("BASE TABLE"); got != types.DatasetKindTable {
		t.Fatalf("expected table, got %s", got)
	}
}

func TestParseDatasetLocation_InvalidLocation(t *testing.T) {
	t.Parallel()

	schema, dataset, err := parseDatasetLocation("public.people")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema != "public" || dataset != "people" {
		t.Fatalf("unexpected result: %s %s", schema, dataset)
	}

	if _, _, err := parseDatasetLocation("people"); err == nil {
		t.Fatal("expected error for invalid location")
	}
}

func TestQualifyNameAndQuoteIdent(t *testing.T) {
	t.Parallel()

	if got := quoteIdent(`ab"cd`); got != `"ab""cd"` {
		t.Fatalf("unexpected quoted ident: %s", got)
	}
	if got := qualifyName("public", "people"); got != `"public"."people"` {
		t.Fatalf("unexpected qualified name: %s", got)
	}
}

func TestNullableJSONString_NilAndValueCases(t *testing.T) {
	t.Parallel()

	if got := nullableJSONString(nil); got != nil {
		t.Fatalf("expected nil, got %s", string(got))
	}
	nullString := "null"
	if got := nullableJSONString(&nullString); got != nil {
		t.Fatalf("expected nil for null string, got %s", string(got))
	}
	value := `"abc"`
	if got := string(nullableJSONString(&value)); got != `"abc"` {
		t.Fatalf("unexpected json value: %s", got)
	}
}
