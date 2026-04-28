package report

import (
	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
	"github.com/EdOoO21/metadata-parser/internal/application/sensitive"
)

type SensitiveField struct {
	SourceName  string
	DatasetName string
	ColumnName  string
	Category    string
	Reason      string
}

func detectSensitiveFields(rows []appports.ReportRow) []SensitiveField {
	fields := make([]SensitiveField, 0)

	for _, row := range rows {
		match, ok := sensitive.MatchField(row.ColumnName, derefString(row.ColumnComment))
		if !ok {
			continue
		}

		fields = append(fields, SensitiveField{
			SourceName:  row.SourceName,
			DatasetName: row.DatasetName,
			ColumnName:  row.ColumnName,
			Category:    match.Category,
			Reason:      match.Reason,
		})
	}

	return fields
}
