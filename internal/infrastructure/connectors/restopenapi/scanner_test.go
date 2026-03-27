package restopenapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EdOoO21/metadata-parser/internal/domain/types"
	"github.com/EdOoO21/metadata-parser/internal/settings"
)

func TestScannerParseSource_ProfilesEndpoint(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"paths": {
				"/users": {
					"get": {
						"summary": "Users",
						"responses": {
							"200": {
								"content": {
									"application/json": {
										"schema": {
											"type": "array",
											"items": {
												"type": "object",
												"required": ["id", "name"],
												"properties": {
													"id": {"type": "integer"},
													"name": {"type": "string"},
													"created_at": {"type": "string", "format": "date-time"}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}`)
	})

	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{"id": 1, "name": "Alice", "created_at": "2026-03-27T10:00:00Z"},
			{"id": 2, "name": "Bob", "created_at": "2026-03-28T10:00:00Z"}
		]`)
	})

	scanner := NewScanner()
	result, err := scanner.ParseSource(context.Background(), settings.SourceConfig{
		Name: "demo-api",
		Kind: "rest",
		Config: settings.SourceConfigDetails{
			BaseURL: server.URL,
			Discovery: &settings.DiscoveryConfig{
				Mode:       "openapi",
				OpenAPIURL: server.URL + "/openapi.json",
			},
		},
	})
	if err != nil {
		t.Fatalf("ParseSource returned error: %v", err)
	}

	if len(result.Datasets) != 1 {
		t.Fatalf("expected 1 dataset, got %d", len(result.Datasets))
	}

	dataset := result.Datasets[0]
	if dataset.Dataset.ProfileStatus != types.ProfileStatusProfiled {
		t.Fatalf("expected profiled status, got %s", dataset.Dataset.ProfileStatus)
	}
	if dataset.Dataset.RowCount == nil || *dataset.Dataset.RowCount != 2 {
		t.Fatalf("expected row count 2, got %+v", dataset.Dataset.RowCount)
	}
	if len(dataset.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(dataset.Columns))
	}
	if dataset.Columns[0].Stat == nil {
		t.Fatal("expected column stats to be present")
	}
}
