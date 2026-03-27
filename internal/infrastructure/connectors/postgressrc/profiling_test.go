package postgressrc

import "testing"

func TestParseDatasetLocation(t *testing.T) {
	t.Parallel()

	schemaName, datasetName, err := parseDatasetLocation("public.users")
	if err != nil {
		t.Fatalf("parseDatasetLocation returned error: %v", err)
	}
	if schemaName != "public" {
		t.Fatalf("expected schema public, got %q", schemaName)
	}
	if datasetName != "users" {
		t.Fatalf("expected dataset users, got %q", datasetName)
	}
}

func TestQuoteIdent(t *testing.T) {
	t.Parallel()

	got := quoteIdent(`my"table`)
	want := `"my""table"`
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestNullableJSONString(t *testing.T) {
	t.Parallel()

	value := `"abc"`
	got := nullableJSONString(&value)
	if string(got) != `"abc"` {
		t.Fatalf("expected json bytes to be preserved, got %q", string(got))
	}

	nullValue := "null"
	if got := nullableJSONString(&nullValue); got != nil {
		t.Fatalf("expected nil for null json, got %q", string(got))
	}
}
