package filescsv

import (
	"context"
	"strings"
	"testing"
)

func TestCSVConnector_Read_WithHeader(t *testing.T) {
	t.Parallel()

	input := "id,name\n1,Alice\n2,Bob\n"

	connector := NewCSVConnector()
	result, err := connector.Read(context.Background(), strings.NewReader(input), DefaultCSVReadOptions())
	if err != nil {
		t.Fatalf("read returned error: %v", err)
	}

	if len(result.Headers) != 2 {
		t.Fatalf("unexpected headers len: %d", len(result.Headers))
	}
	if result.Headers[0] != "id" || result.Headers[1] != "name" {
		t.Fatalf("unexpected headers: %#v", result.Headers)
	}
	if len(result.Rows) != 2 {
		t.Fatalf("unexpected rows len: %d", len(result.Rows))
	}
	if result.Rows[0].Values["id"] != "1" || result.Rows[0].Values["name"] != "Alice" {
		t.Fatalf("unexpected first row: %#v", result.Rows[0].Values)
	}
}

func TestCSVConnector_Read_WithoutHeader(t *testing.T) {
	t.Parallel()

	input := "1,Alice\n2,Bob\n"

	connector := NewCSVConnector()
	opts := DefaultCSVReadOptions()
	opts.HasHeaderRecord = false

	result, err := connector.Read(context.Background(), strings.NewReader(input), opts)
	if err != nil {
		t.Fatalf("read returned error: %v", err)
	}

	if len(result.Headers) != 2 {
		t.Fatalf("unexpected headers len: %d", len(result.Headers))
	}
	if result.Headers[0] != "column_1" || result.Headers[1] != "column_2" {
		t.Fatalf("unexpected generated headers: %#v", result.Headers)
	}
	if result.Rows[1].Values["column_2"] != "Bob" {
		t.Fatalf("unexpected second row: %#v", result.Rows[1].Values)
	}
}

func TestCSVConnector_Read_DuplicateHeaders(t *testing.T) {
	t.Parallel()

	input := "id,id\n1,Alice\n"

	connector := NewCSVConnector()
	_, err := connector.Read(context.Background(), strings.NewReader(input), DefaultCSVReadOptions())
	if err == nil {
		t.Fatalf("expected duplicate headers error")
	}
}
