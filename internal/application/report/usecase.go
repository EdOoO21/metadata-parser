package report

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
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
	Markdown string
	CSV      []byte
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

	return &Result{
		Markdown: renderMarkdown(run, rows),
		CSV:      renderCSV(rows),
	}, nil
}

func renderMarkdown(run *model.Run, rows []appports.ReportRow) string {
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
			b.WriteString("\n")
			b.WriteString("| # | Column | Original Type | Normalized Type | Nullable | Comment |\n")
			b.WriteString("|---|---|---|---|---|---|\n")
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
