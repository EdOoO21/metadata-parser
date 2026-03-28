package report

import (
	"testing"

	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
)

func TestDetectSensitiveFields(t *testing.T) {
	t.Parallel()

	rows := reportRowsForSensitiveTest{
		{sourceName: "demo", datasetName: "people", columnName: "fio"},
		{sourceName: "demo", datasetName: "people", columnName: "email_address"},
		{sourceName: "demo", datasetName: "people", columnName: "doc", columnComment: stringPtr("passport number")},
		{sourceName: "demo", datasetName: "people", columnName: "age"},
	}

	fields := detectSensitiveFields(rows.toReportRows())

	if len(fields) != 3 {
		t.Fatalf("expected 3 sensitive fields, got %d", len(fields))
	}

	if fields[0].Category != "person_name" {
		t.Fatalf("unexpected first category: %+v", fields[0])
	}
	if fields[1].Category != "contact" {
		t.Fatalf("unexpected second category: %+v", fields[1])
	}
	if fields[2].Category != "government_id" {
		t.Fatalf("unexpected third category: %+v", fields[2])
	}
}

type reportRowForSensitiveTest struct {
	sourceName    string
	datasetName   string
	columnName    string
	columnComment *string
}

type reportRowsForSensitiveTest []reportRowForSensitiveTest

func (rows reportRowsForSensitiveTest) toReportRows() []appports.ReportRow {
	result := make([]appports.ReportRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, appports.ReportRow{
			SourceName:    row.sourceName,
			DatasetName:   row.datasetName,
			ColumnName:    row.columnName,
			ColumnComment: row.columnComment,
		})
	}
	return result
}

func stringPtr(value string) *string {
	return &value
}
