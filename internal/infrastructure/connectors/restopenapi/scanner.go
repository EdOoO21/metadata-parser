package restopenapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/EdOoO21/metadata-parser/internal/application/contracts"
	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"
	"github.com/EdOoO21/metadata-parser/internal/infrastructure/connectors/shared"
	"github.com/EdOoO21/metadata-parser/internal/settings"
)

type Scanner struct {
	client *http.Client
}

const defaultRESTMaxResponseBytes int64 = 10 * 1024 * 1024

func NewScanner() *Scanner {
	return &Scanner{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *Scanner) ParseSource(ctx context.Context, src settings.SourceConfig) (*contracts.SourceScanResult, error) {
	effectiveConfigJSON, err := json.Marshal(src.Config)
	if err != nil {
		return nil, fmt.Errorf("marshal effective config: %w", err)
	}

	maxResponseBytes := effectiveRESTMaxResponseBytes(src.Config.MaxResponseBytes)
	spec, err := s.loadSpec(ctx, src.Config.Discovery.OpenAPIURL, maxResponseBytes)
	if err != nil {
		return nil, err
	}

	datasets := make([]contracts.ScannedDataset, 0, len(spec.Paths))
	for path, item := range spec.Paths {
		for _, endpoint := range item.operations() {
			schema := endpoint.operation.responseSchema(spec.Components)
			columns := buildColumnsFromSchema(resolveResponseSchema(schema, spec.Components), spec.Components)

			comment := firstNonEmpty(endpoint.operation.Summary, endpoint.operation.Description)
			var commentPtr *string
			if comment != "" {
				commentPtr = &comment
			}

			metadataJSON, err := buildEndpointMetadataJSON(endpoint.method, path, item.Parameters, endpoint.operation)
			if err != nil {
				return nil, fmt.Errorf("marshal endpoint metadata: %w", err)
			}

			dataset := contracts.ScannedDataset{
				Dataset: model.Dataset{
					Kind:          types.DatasetKindEndpoint,
					DatasetKey:    endpoint.method + ":" + path,
					Name:          endpoint.method + " " + path,
					Location:      joinURL(src.Config.BaseURL, path),
					Comment:       commentPtr,
					DiscoveredAt:  time.Now().UTC(),
					ProfileStatus: types.ProfileStatusDiscoveredOnly,
					MetadataJSON:  metadataJSON,
				},
				Columns: columns,
			}

			if endpoint.method != http.MethodGet {
				datasets = append(datasets, dataset)
				continue
			}

			if strings.Contains(path, "{") {
				dataset.Dataset.ProfileStatus = types.ProfileStatusSkippedRequiresParams
			} else if err := s.profileEndpoint(ctx, joinURL(src.Config.BaseURL, path), &dataset, maxResponseBytes); err != nil {
				errMessage := err.Error()
				dataset.Dataset.ProfileStatus = types.ProfileStatusFailed
				dataset.Dataset.ProfileError = &errMessage
			}

			datasets = append(datasets, dataset)
		}
	}

	return &contracts.SourceScanResult{
		Source: model.Source{
			Name: src.Name,
			Kind: types.SourceKindREST,
		},
		EffectiveConfigJSON: effectiveConfigJSON,
		Datasets:            datasets,
	}, nil
}

func (s *Scanner) loadSpec(ctx context.Context, url string, maxResponseBytes int64) (*openAPISpec, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build openapi request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch openapi spec: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch openapi spec: unexpected status %d", resp.StatusCode)
	}

	var spec openAPISpec
	if err := decodeLimitedJSON(resp.Body, maxResponseBytes, &spec); err != nil {
		return nil, fmt.Errorf("decode openapi spec: %w", err)
	}

	return &spec, nil
}

func effectiveRESTMaxResponseBytes(value int64) int64 {
	if value > 0 {
		return value
	}
	return defaultRESTMaxResponseBytes
}

func decodeLimitedJSON(body io.Reader, maxResponseBytes int64, target any) error {
	limit := effectiveRESTMaxResponseBytes(maxResponseBytes)
	data, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return fmt.Errorf("read json response: %w", err)
	}
	if int64(len(data)) > limit {
		return fmt.Errorf("json response exceeds max_response_bytes limit %d", limit)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return err
	}
	return nil
}

func buildColumnsFromSchema(schema *openAPISchema, components openAPIComponents) []contracts.ScannedColumn {
	if schema == nil {
		return nil
	}

	schema = resolveResponseSchema(schema, components)

	required := make(map[string]struct{}, len(schema.Required))
	for _, name := range schema.Required {
		required[name] = struct{}{}
	}

	names := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		names = append(names, name)
	}
	sortStrings(names)

	columns := make([]contracts.ScannedColumn, 0, len(names))
	for i, name := range names {
		property := resolveResponseSchema(schema.Properties[name], components)
		if property == nil {
			continue
		}

		comment := firstNonEmpty(property.Title, property.Description)
		var commentPtr *string
		if comment != "" {
			commentPtr = &comment
		}

		_, isRequired := required[name]
		columns = append(columns, contracts.ScannedColumn{
			Column: model.Column{
				Name:            name,
				OriginalType:    firstNonEmpty(property.Type, inferTypeFromSchema(property)),
				NormalizedType:  normalizeRESTSchemaType(property),
				IsNullable:      !isRequired || property.Nullable,
				Comment:         commentPtr,
				OrdinalPosition: i + 1,
			},
		})
	}

	return columns
}

func buildEndpointMetadataJSON(method string, path string, pathParameters []openAPIParameter, operation *openAPIOperation) ([]byte, error) {
	metadata := map[string]any{
		"method":      method,
		"path":        path,
		"summary":     operation.Summary,
		"description": operation.Description,
		"operationId": operation.OperationID,
		"tags":        operation.Tags,
		"parameters":  append(append([]openAPIParameter{}, pathParameters...), operation.Parameters...),
		"requestBody": operation.RequestBody,
		"responses":   operation.Responses,
	}

	return json.Marshal(metadata)
}

func resolveResponseSchema(schema *openAPISchema, components openAPIComponents) *openAPISchema {
	if schema == nil {
		return nil
	}

	if schema.Ref == "" {
		if schema.Type == "array" && schema.Items != nil {
			resolvedItems := resolveResponseSchema(schema.Items, components)
			if resolvedItems != nil && resolvedItems.Type == "object" {
				return resolvedItems
			}
		}
		return schema
	}

	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(schema.Ref, prefix) {
		return schema
	}

	name := strings.TrimPrefix(schema.Ref, prefix)
	resolved := components.Schemas[name]
	if resolved == nil {
		return schema
	}

	return resolveResponseSchema(resolved, components)
}

func inferTypeFromSchema(schema *openAPISchema) string {
	if schema == nil {
		return "string"
	}
	if schema.Type != "" {
		return schema.Type
	}
	if schema.Items != nil {
		return "array"
	}
	if len(schema.Properties) > 0 {
		return "object"
	}
	return "string"
}

func normalizeRESTSchemaType(schema *openAPISchema) types.CanonicalType {
	if schema == nil {
		return types.CanonicalTypeString
	}

	sourceType := firstNonEmpty(schema.Type, inferTypeFromSchema(schema))
	if sourceType == "string" {
		switch strings.TrimSpace(strings.ToLower(schema.Format)) {
		case "date", "date-time":
			return types.CanonicalTypeTimestamp
		}
	}

	return shared.NormalizeType(sourceType)
}

func joinURL(baseURL, path string) string {
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func sortStrings(values []string) {
	for i := range values {
		for j := i + 1; j < len(values); j++ {
			if values[j] < values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}

type openAPISpec struct {
	Paths      map[string]openAPIPathItem `json:"paths"`
	Components openAPIComponents          `json:"components"`
}

type openAPIComponents struct {
	Schemas map[string]*openAPISchema `json:"schemas"`
}

type openAPIPathItem struct {
	Parameters []openAPIParameter `json:"parameters"`
	Get        *openAPIOperation  `json:"get"`
	Post       *openAPIOperation  `json:"post"`
	Put        *openAPIOperation  `json:"put"`
	Patch      *openAPIOperation  `json:"patch"`
	Delete     *openAPIOperation  `json:"delete"`
	Head       *openAPIOperation  `json:"head"`
	Options    *openAPIOperation  `json:"options"`
	Trace      *openAPIOperation  `json:"trace"`
}

type openAPIEndpointOperation struct {
	method    string
	operation *openAPIOperation
}

func (i openAPIPathItem) operations() []openAPIEndpointOperation {
	candidates := []openAPIEndpointOperation{
		{method: http.MethodGet, operation: i.Get},
		{method: http.MethodPost, operation: i.Post},
		{method: http.MethodPut, operation: i.Put},
		{method: http.MethodPatch, operation: i.Patch},
		{method: http.MethodDelete, operation: i.Delete},
		{method: http.MethodHead, operation: i.Head},
		{method: http.MethodOptions, operation: i.Options},
		{method: http.MethodTrace, operation: i.Trace},
	}

	operations := make([]openAPIEndpointOperation, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.operation != nil {
			operations = append(operations, candidate)
		}
	}

	return operations
}

type openAPIOperation struct {
	Tags        []string                      `json:"tags"`
	Summary     string                        `json:"summary"`
	Description string                        `json:"description"`
	OperationID string                        `json:"operationId"`
	Parameters  []openAPIParameter            `json:"parameters"`
	RequestBody *openAPIRequestBodyRef        `json:"requestBody"`
	Responses   map[string]openAPIResponseRef `json:"responses"`
}

func (o *openAPIOperation) responseSchema(components openAPIComponents) *openAPISchema {
	for _, status := range []string{"200", "201", "default"} {
		response, ok := o.Responses[status]
		if !ok || response.Content == nil {
			continue
		}

		if media, ok := response.Content["application/json"]; ok {
			return resolveResponseSchema(media.Schema, components)
		}
	}

	return nil
}

type openAPIResponseRef struct {
	Description string                      `json:"description"`
	Content     map[string]openAPIMediaType `json:"content"`
}

type openAPIParameter struct {
	Name        string         `json:"name"`
	In          string         `json:"in"`
	Description string         `json:"description"`
	Required    bool           `json:"required"`
	Schema      *openAPISchema `json:"schema"`
}

type openAPIRequestBodyRef struct {
	Description string                      `json:"description"`
	Required    bool                        `json:"required"`
	Content     map[string]openAPIMediaType `json:"content"`
}

type openAPIMediaType struct {
	Schema *openAPISchema `json:"schema"`
}

type openAPISchema struct {
	Ref         string                    `json:"$ref"`
	Title       string                    `json:"title"`
	Description string                    `json:"description"`
	Type        string                    `json:"type"`
	Format      string                    `json:"format"`
	Nullable    bool                      `json:"nullable"`
	Required    []string                  `json:"required"`
	Properties  map[string]*openAPISchema `json:"properties"`
	Items       *openAPISchema            `json:"items"`
}

var _ appports.SourceScanner = (*Scanner)(nil)
