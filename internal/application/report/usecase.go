package report

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"sort"
	"strconv"
	"strings"
	"time"

	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
	"github.com/EdOoO21/metadata-parser/internal/domain/model"
)

type ExecuteInput struct {
	Repository appports.CatalogRepository
	RunID      int64
}

type Result struct {
	Markdown        string
	HTML            string
	CSV             []byte
	SensitiveFields []SensitiveField
}

type ReportCatalogUseCase struct{}

func NewReportCatalogUseCase() *ReportCatalogUseCase {
	return &ReportCatalogUseCase{}
}

func (uc *ReportCatalogUseCase) Execute(ctx context.Context, input ExecuteInput) (*Result, error) {
	if input.Repository == nil {
		return nil, fmt.Errorf("catalog repository is not configured")
	}
	if input.RunID <= 0 {
		return nil, fmt.Errorf("run id must be greater than zero")
	}

	run, err := input.Repository.GetRun(ctx, input.RunID)
	if err != nil {
		return nil, fmt.Errorf("get run: %w", err)
	}

	rows, err := input.Repository.ListReportRows(ctx, input.RunID)
	if err != nil {
		return nil, fmt.Errorf("list report rows: %w", err)
	}

	sensitiveFields := detectSensitiveFields(rows)

	return &Result{
		Markdown:        renderMarkdown(run, rows, sensitiveFields),
		HTML:            renderHTML(run, rows, sensitiveFields),
		CSV:             renderCSV(rows),
		SensitiveFields: sensitiveFields,
	}, nil
}

func renderHTML(run *model.Run, rows []appports.ReportRow, sensitiveFields []SensitiveField) string {
	var b strings.Builder

	b.WriteString("<!doctype html>\n")
	b.WriteString("<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	b.WriteString("<title>Catalog Report</title>\n")
	b.WriteString("<style>")
	b.WriteString("body{font-family:system-ui,-apple-system,sans-serif;max-width:1100px;margin:40px auto;padding:0 16px;line-height:1.5;color:#111;}h1,h2,h3{line-height:1.2;}table{border-collapse:collapse;width:100%;margin:16px 0 32px;}th,td{border:1px solid #d0d7de;padding:8px 10px;text-align:left;vertical-align:top;}th{background:#f6f8fa;}code{background:#f6f8fa;padding:2px 6px;border-radius:4px;}ul{padding-left:20px;}li{margin:4px 0;}")
	b.WriteString("</style>\n</head>\n<body>\n")

	fmt.Fprintf(&b, "<h1>Catalog Report for Run %d</h1>\n", run.ID)
	b.WriteString("<ul>\n")
	fmt.Fprintf(&b, "<li>Status: <code>%s</code></li>\n", html.EscapeString(string(run.Status)))
	fmt.Fprintf(&b, "<li>Started at: <code>%s</code></li>\n", html.EscapeString(formatTime(run.StartedAt)))
	if run.FinishedAt != nil {
		fmt.Fprintf(&b, "<li>Finished at: <code>%s</code></li>\n", html.EscapeString(formatTime(*run.FinishedAt)))
	}
	if run.ErrorMessage != nil && strings.TrimSpace(*run.ErrorMessage) != "" {
		fmt.Fprintf(&b, "<li>Error: <code>%s</code></li>\n", html.EscapeString(*run.ErrorMessage))
	}
	b.WriteString("</ul>\n")

	if len(rows) == 0 {
		b.WriteString("<p>Датасеты для этого запуска не найдены.</p>\n</body>\n</html>\n")
		return b.String()
	}

	writeSensitiveFieldsHTML(&b, sensitiveFields)

	currentSource := ""
	currentDataset := ""
	tableOpen := false

	for _, row := range rows {
		if row.SourceName != currentSource {
			if tableOpen {
				b.WriteString("</tbody>\n</table>\n")
				tableOpen = false
				currentDataset = ""
			}
			currentSource = row.SourceName
			fmt.Fprintf(&b, "<h2>Source: %s (<code>%s</code>)</h2>\n", html.EscapeString(row.SourceName), html.EscapeString(string(row.SourceKind)))
		}

		datasetKey := row.SourceName + "::" + row.DatasetKey
		if datasetKey != currentDataset {
			if tableOpen {
				b.WriteString("</tbody>\n</table>\n")
				tableOpen = false
			}
			currentDataset = datasetKey

			fmt.Fprintf(&b, "<h3>Dataset: %s (<code>%s</code>)</h3>\n", html.EscapeString(row.DatasetName), html.EscapeString(string(row.DatasetKind)))
			b.WriteString("<ul>\n")
			fmt.Fprintf(&b, "<li>Location: <code>%s</code></li>\n", html.EscapeString(row.DatasetLocation))
			if row.DatasetRowCount != nil {
				fmt.Fprintf(&b, "<li>Row count: <code>%d</code></li>\n", *row.DatasetRowCount)
			}
			fmt.Fprintf(&b, "<li>Profile status: <code>%s</code></li>\n", html.EscapeString(string(row.DatasetProfileStatus)))
			if row.DatasetComment != nil && strings.TrimSpace(*row.DatasetComment) != "" {
				fmt.Fprintf(&b, "<li>Comment: %s</li>\n", html.EscapeString(*row.DatasetComment))
			}
			writeEndpointMetadataHTML(&b, row)
			b.WriteString("</ul>\n")
			if !row.ColumnPresent {
				b.WriteString("<p>Columns were not discovered for this dataset.</p>\n")
			}
		}

		if !row.ColumnPresent {
			continue
		}

		if !tableOpen {
			b.WriteString("<table>\n<thead><tr><th>#</th><th>Column</th><th>Original Type</th><th>Normalized Type</th><th>Nullable</th><th>Comment</th></tr></thead>\n<tbody>\n")
			tableOpen = true
		}

		fmt.Fprintf(
			&b,
			"<tr><td>%d</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>\n",
			row.ColumnOrdinal,
			html.EscapeString(row.ColumnName),
			html.EscapeString(row.ColumnOriginalType),
			html.EscapeString(string(row.ColumnNormalizedType)),
			html.EscapeString(formatBool(row.ColumnIsNullable)),
			html.EscapeString(derefString(row.ColumnComment)),
		)
	}

	if tableOpen {
		b.WriteString("</tbody>\n</table>\n")
	}

	b.WriteString("</body>\n</html>\n")
	return b.String()
}

func renderMarkdown(run *model.Run, rows []appports.ReportRow, sensitiveFields []SensitiveField) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# Catalog Report for Run %d\n\n", run.ID)
	fmt.Fprintf(&b, "- Status: `%s`\n", run.Status)
	fmt.Fprintf(&b, "- Started at: `%s`\n", formatTime(run.StartedAt))
	if run.FinishedAt != nil {
		fmt.Fprintf(&b, "- Finished at: `%s`\n", formatTime(*run.FinishedAt))
	}
	if run.ErrorMessage != nil && strings.TrimSpace(*run.ErrorMessage) != "" {
		fmt.Fprintf(&b, "- Error: `%s`\n", *run.ErrorMessage)
	}
	b.WriteString("\n")

	if len(rows) == 0 {
		b.WriteString("Датасеты для этого запуска не найдены.\n")
		return b.String()
	}

	writeSensitiveFieldsMarkdown(&b, sensitiveFields)

	currentSource := ""
	currentDataset := ""

	for _, row := range rows {
		if row.SourceName != currentSource {
			currentSource = row.SourceName
			currentDataset = ""

			fmt.Fprintf(&b, "## Source: %s (`%s`)\n\n", row.SourceName, row.SourceKind)
		}

		datasetKey := row.SourceName + "::" + row.DatasetKey
		if datasetKey != currentDataset {
			currentDataset = datasetKey

			b.WriteString(fmt.Sprintf("### Dataset: %s (`%s`)\n\n", row.DatasetName, row.DatasetKind))
			b.WriteString(fmt.Sprintf("- Location: `%s`\n", row.DatasetLocation))
			if row.DatasetRowCount != nil {
				b.WriteString(fmt.Sprintf("- Row count: `%d`\n", *row.DatasetRowCount))
			}
			b.WriteString(fmt.Sprintf("- Profile status: `%s`\n", row.DatasetProfileStatus))
			if row.DatasetComment != nil && strings.TrimSpace(*row.DatasetComment) != "" {
				b.WriteString(fmt.Sprintf("- Comment: %s\n", *row.DatasetComment))
			}
			b.WriteString(renderEndpointMetadataMarkdown(row))
			b.WriteString("\n")
			if row.ColumnPresent {
				b.WriteString("| # | Column | Original Type | Normalized Type | Nullable | Comment |\n")
				b.WriteString("|---|---|---|---|---|---|\n")
			} else {
				b.WriteString("_Columns were not discovered for this dataset._\n\n")
			}
		}

		if !row.ColumnPresent {
			continue
		}

		b.WriteString(fmt.Sprintf(
			"| %d | %s | %s | %s | %s | %s |\n",
			row.ColumnOrdinal,
			escapeMarkdownCell(row.ColumnName),
			escapeMarkdownCell(row.ColumnOriginalType),
			escapeMarkdownCell(string(row.ColumnNormalizedType)),
			formatBool(row.ColumnIsNullable),
			escapeMarkdownCell(derefString(row.ColumnComment)),
		))
	}

	return b.String()
}

func writeSensitiveFieldsHTML(b *strings.Builder, sensitiveFields []SensitiveField) {
	if len(sensitiveFields) == 0 {
		b.WriteString("<h2>Potentially Sensitive Fields</h2>\n<p>Потенциально чувствительные поля не найдены.</p>\n")
		return
	}

	b.WriteString("<h2>Potentially Sensitive Fields</h2>\n")
	b.WriteString("<table>\n<thead><tr><th>Source</th><th>Dataset</th><th>Column</th><th>Category</th><th>Reason</th></tr></thead>\n<tbody>\n")
	for _, field := range sensitiveFields {
		fmt.Fprintf(
			b,
			"<tr><td>%s</td><td>%s</td><td>%s</td><td><code>%s</code></td><td>%s</td></tr>\n",
			html.EscapeString(field.SourceName),
			html.EscapeString(field.DatasetName),
			html.EscapeString(field.ColumnName),
			html.EscapeString(field.Category),
			html.EscapeString(field.Reason),
		)
	}
	b.WriteString("</tbody>\n</table>\n")
}

func writeSensitiveFieldsMarkdown(b *strings.Builder, sensitiveFields []SensitiveField) {
	b.WriteString("## Potentially Sensitive Fields\n\n")
	if len(sensitiveFields) == 0 {
		b.WriteString("Потенциально чувствительные поля не найдены.\n\n")
		return
	}

	b.WriteString("| Source | Dataset | Column | Category | Reason |\n")
	b.WriteString("|---|---|---|---|---|\n")
	for _, field := range sensitiveFields {
		b.WriteString(fmt.Sprintf(
			"| %s | %s | %s | %s | %s |\n",
			escapeMarkdownCell(field.SourceName),
			escapeMarkdownCell(field.DatasetName),
			escapeMarkdownCell(field.ColumnName),
			escapeMarkdownCell(field.Category),
			escapeMarkdownCell(field.Reason),
		))
	}
	b.WriteString("\n")
}

type endpointReportMetadata struct {
	Method      string                     `json:"method"`
	Path        string                     `json:"path"`
	OperationID string                     `json:"operationId"`
	Tags        []string                   `json:"tags"`
	Parameters  []endpointReportParameter  `json:"parameters"`
	RequestBody json.RawMessage            `json:"requestBody"`
	Responses   map[string]json.RawMessage `json:"responses"`
}

type endpointReportParameter struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Required bool   `json:"required"`
}

func parseEndpointMetadata(row appports.ReportRow) (endpointReportMetadata, bool) {
	if row.DatasetKind != "endpoint" || len(row.DatasetMetadataJSON) == 0 {
		return endpointReportMetadata{}, false
	}

	var metadata endpointReportMetadata
	if err := json.Unmarshal(row.DatasetMetadataJSON, &metadata); err != nil {
		return endpointReportMetadata{}, false
	}

	if strings.TrimSpace(metadata.Method) == "" && strings.TrimSpace(metadata.Path) == "" {
		return endpointReportMetadata{}, false
	}

	return metadata, true
}

func renderEndpointMetadataMarkdown(row appports.ReportRow) string {
	metadata, ok := parseEndpointMetadata(row)
	if !ok {
		return ""
	}

	var b strings.Builder
	if metadata.Method != "" && metadata.Path != "" {
		b.WriteString(fmt.Sprintf("- Endpoint: `%s %s`\n", metadata.Method, metadata.Path))
	}
	if metadata.OperationID != "" {
		b.WriteString(fmt.Sprintf("- Operation ID: `%s`\n", metadata.OperationID))
	}
	if len(metadata.Tags) > 0 {
		b.WriteString(fmt.Sprintf("- Tags: `%s`\n", strings.Join(metadata.Tags, "`, `")))
	}
	if len(metadata.Parameters) > 0 {
		b.WriteString(fmt.Sprintf("- Parameters: %s\n", formatEndpointParameters(metadata.Parameters)))
	}
	if len(metadata.RequestBody) > 0 && string(metadata.RequestBody) != "null" {
		b.WriteString("- Request body: `yes`\n")
	}
	if len(metadata.Responses) > 0 {
		b.WriteString(fmt.Sprintf("- Responses: `%s`\n", strings.Join(sortedMapKeys(metadata.Responses), "`, `")))
	}
	return b.String()
}

func writeEndpointMetadataHTML(b *strings.Builder, row appports.ReportRow) {
	metadata, ok := parseEndpointMetadata(row)
	if !ok {
		return
	}

	if metadata.Method != "" && metadata.Path != "" {
		fmt.Fprintf(b, "<li>Endpoint: <code>%s %s</code></li>\n", html.EscapeString(metadata.Method), html.EscapeString(metadata.Path))
	}
	if metadata.OperationID != "" {
		fmt.Fprintf(b, "<li>Operation ID: <code>%s</code></li>\n", html.EscapeString(metadata.OperationID))
	}
	if len(metadata.Tags) > 0 {
		fmt.Fprintf(b, "<li>Tags: <code>%s</code></li>\n", html.EscapeString(strings.Join(metadata.Tags, ", ")))
	}
	if len(metadata.Parameters) > 0 {
		fmt.Fprintf(b, "<li>Parameters: %s</li>\n", html.EscapeString(formatEndpointParameters(metadata.Parameters)))
	}
	if len(metadata.RequestBody) > 0 && string(metadata.RequestBody) != "null" {
		b.WriteString("<li>Request body: <code>yes</code></li>\n")
	}
	if len(metadata.Responses) > 0 {
		fmt.Fprintf(b, "<li>Responses: <code>%s</code></li>\n", html.EscapeString(strings.Join(sortedMapKeys(metadata.Responses), ", ")))
	}
}

func formatEndpointParameters(parameters []endpointReportParameter) string {
	values := make([]string, 0, len(parameters))
	for _, parameter := range parameters {
		required := "optional"
		if parameter.Required {
			required = "required"
		}
		values = append(values, fmt.Sprintf("%s %s (%s)", parameter.In, parameter.Name, required))
	}
	return strings.Join(values, ", ")
}

func sortedMapKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func renderCSV(rows []appports.ReportRow) []byte {
	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)

	_ = writer.Write([]string{
		"source_name",
		"source_kind",
		"dataset_name",
		"dataset_kind",
		"dataset_key",
		"dataset_location",
		"dataset_comment",
		"dataset_row_count",
		"dataset_profile_status",
		"dataset_metadata_json",
		"column_ordinal",
		"column_name",
		"column_original_type",
		"column_normalized_type",
		"column_is_nullable",
		"column_comment",
	})

	for _, row := range rows {
		record := []string{
			row.SourceName,
			string(row.SourceKind),
			row.DatasetName,
			string(row.DatasetKind),
			row.DatasetKey,
			row.DatasetLocation,
			derefString(row.DatasetComment),
			formatInt64Ptr(row.DatasetRowCount),
			string(row.DatasetProfileStatus),
			string(row.DatasetMetadataJSON),
			strconv.Itoa(row.ColumnOrdinal),
			row.ColumnName,
			row.ColumnOriginalType,
			string(row.ColumnNormalizedType),
			strconv.FormatBool(row.ColumnIsNullable),
			derefString(row.ColumnComment),
		}
		_ = writer.Write(record)
	}

	writer.Flush()
	return buffer.Bytes()
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func formatInt64Ptr(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}

func formatBool(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func escapeMarkdownCell(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "|", "\\|")
	return value
}
