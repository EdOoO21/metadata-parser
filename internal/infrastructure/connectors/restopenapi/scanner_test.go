package restopenapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EdOoO21/metadata-parser/internal/application/contracts"
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
						"operationId": "listUsers",
						"parameters": [
							{
								"name": "limit",
								"in": "query",
								"required": false,
								"schema": {"type": "integer"}
							}
						],
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
					},
					"post": {
						"summary": "Create user",
						"operationId": "createUser",
						"requestBody": {
							"required": true,
							"content": {
								"application/json": {
									"schema": {
										"type": "object",
										"required": ["name"],
										"properties": {
											"name": {"type": "string"}
										}
									}
								}
							}
						},
						"responses": {
							"201": {
								"content": {
									"application/json": {
										"schema": {
											"type": "object",
											"required": ["id", "name"],
											"properties": {
												"id": {"type": "integer"},
												"name": {"type": "string"}
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

	if len(result.Datasets) != 2 {
		t.Fatalf("expected 2 datasets, got %d", len(result.Datasets))
	}

	dataset := result.Datasets[0]
	if dataset.Dataset.Name != "GET /users" {
		t.Fatalf("unexpected first dataset: %s", dataset.Dataset.Name)
	}
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
	var createdAtFound bool
	for _, column := range dataset.Columns {
		if column.Column.Name != "created_at" {
			continue
		}
		createdAtFound = true
		if column.Column.NormalizedType != types.CanonicalTypeTimestamp {
			t.Fatalf("expected created_at to be TIMESTAMP, got %s", column.Column.NormalizedType)
		}
		if column.Stat == nil || column.Stat.MinValueJSON == nil || column.Stat.MaxValueJSON == nil {
			t.Fatalf("expected timestamp stats for created_at, got %+v", column.Stat)
		}
	}
	if !createdAtFound {
		t.Fatal("expected created_at column to be present")
	}

	postDataset := result.Datasets[1]
	if postDataset.Dataset.Name != "POST /users" {
		t.Fatalf("unexpected second dataset: %s", postDataset.Dataset.Name)
	}
	if postDataset.Dataset.ProfileStatus != types.ProfileStatusDiscoveredOnly {
		t.Fatalf("expected post endpoint to be discovered only, got %s", postDataset.Dataset.ProfileStatus)
	}
	if postDataset.Dataset.RowCount != nil {
		t.Fatalf("expected post endpoint row count to be empty, got %+v", postDataset.Dataset.RowCount)
	}
	if len(postDataset.Columns) != 2 {
		t.Fatalf("expected post response columns, got %d", len(postDataset.Columns))
	}

	var metadata map[string]any
	if err := json.Unmarshal(postDataset.Dataset.MetadataJSON, &metadata); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if metadata["method"] != "POST" || metadata["operationId"] != "createUser" {
		t.Fatalf("unexpected post metadata: %+v", metadata)
	}
	if metadata["requestBody"] == nil || metadata["responses"] == nil {
		t.Fatalf("expected requestBody and responses in metadata: %+v", metadata)
	}
}

func TestScannerParseSource_DiscoversAllHTTPMethods(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"paths": {
				"/items/{id}": {
					"parameters": [{"name": "id", "in": "path", "required": true, "schema": {"type": "integer"}}],
					"get": {
						"operationId": "getItem",
						"responses": {"200": {"content": {"application/json": {"schema": {"type": "object", "properties": {"id": {"type": "integer"}}}}}}}
					},
					"post": {
						"operationId": "createItem",
						"requestBody": {"required": true, "content": {"application/json": {"schema": {"type": "object", "properties": {"name": {"type": "string"}}}}}},
						"responses": {"201": {"content": {"application/json": {"schema": {"type": "object", "properties": {"id": {"type": "integer"}}}}}}}
					},
					"put": {
						"operationId": "replaceItem",
						"responses": {"200": {"content": {"application/json": {"schema": {"type": "object", "properties": {"id": {"type": "integer"}}}}}}}
					},
					"patch": {
						"operationId": "patchItem",
						"responses": {"200": {"content": {"application/json": {"schema": {"type": "object", "properties": {"id": {"type": "integer"}}}}}}}
					},
					"delete": {
						"operationId": "deleteItem",
						"responses": {"204": {"description": "Deleted"}}
					},
					"head": {
						"operationId": "headItem",
						"responses": {"200": {"description": "Exists"}}
					},
					"options": {
						"operationId": "optionsItem",
						"responses": {"200": {"description": "Allowed methods"}}
					},
					"trace": {
						"operationId": "traceItem",
						"responses": {"200": {"description": "Trace"}}
					}
				}
			}
		}`)
	})

	scanner := NewScanner()
	result, err := scanner.ParseSource(context.Background(), settings.SourceConfig{
		Name: "all-methods-api",
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

	wantMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "TRACE"}
	if len(result.Datasets) != len(wantMethods) {
		t.Fatalf("expected %d datasets, got %d", len(wantMethods), len(result.Datasets))
	}

	for i, method := range wantMethods {
		dataset := result.Datasets[i]
		if dataset.Dataset.Name != method+" /items/{id}" {
			t.Fatalf("unexpected dataset at %d: %s", i, dataset.Dataset.Name)
		}

		var metadata map[string]any
		if err := json.Unmarshal(dataset.Dataset.MetadataJSON, &metadata); err != nil {
			t.Fatalf("unmarshal metadata for %s: %v", method, err)
		}
		if metadata["method"] != method || metadata["path"] != "/items/{id}" {
			t.Fatalf("unexpected metadata for %s: %+v", method, metadata)
		}
		if _, ok := metadata["parameters"].([]any); !ok {
			t.Fatalf("expected parameters metadata for %s: %+v", method, metadata)
		}
	}

	if result.Datasets[0].Dataset.ProfileStatus != types.ProfileStatusSkippedRequiresParams {
		t.Fatalf("expected GET with path params to be skipped, got %s", result.Datasets[0].Dataset.ProfileStatus)
	}
	if result.Datasets[1].Dataset.ProfileStatus != types.ProfileStatusDiscoveredOnly {
		t.Fatalf("expected POST to be discovered only, got %s", result.Datasets[1].Dataset.ProfileStatus)
	}
	if len(result.Datasets[4].Columns) != 0 {
		t.Fatalf("expected DELETE without response schema to have no columns, got %d", len(result.Datasets[4].Columns))
	}
}

func TestLoadSpecRejectsOversizedResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"paths":{},"padding":"too large"}`)
	}))
	defer server.Close()

	scanner := NewScanner()
	_, err := scanner.loadSpec(context.Background(), server.URL, 10)
	if err == nil {
		t.Fatal("expected oversized OpenAPI response error")
	}
	if !strings.Contains(err.Error(), "max_response_bytes") {
		t.Fatalf("expected max_response_bytes error, got: %v", err)
	}
}

func TestProfileEndpointRejectsOversizedResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"name":"too large"}]`)
	}))
	defer server.Close()

	scanner := NewScanner()
	err := scanner.profileEndpoint(context.Background(), server.URL, &contracts.ScannedDataset{}, 10)
	if err == nil {
		t.Fatal("expected oversized endpoint response error")
	}
	if !strings.Contains(err.Error(), "max_response_bytes") {
		t.Fatalf("expected max_response_bytes error, got: %v", err)
	}
}
