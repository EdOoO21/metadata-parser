package report

import (
	"fmt"
	"strings"

	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
)

type SensitiveField struct {
	SourceName  string
	DatasetName string
	ColumnName  string
	Category    string
	Reason      string
}

type sensitiveRule struct {
	category string
	patterns []string
}

var sensitiveRules = []sensitiveRule{
	{category: "person_name", patterns: []string{"fio", "фио", "full_name", "fullname", "first_name", "last_name", "middle_name", "surname", "family_name", "given_name"}},
	{category: "contact", patterns: []string{"email", "e_mail", "mail", "phone", "mobile", "telephone", "tel"}},
	{category: "government_id", patterns: []string{"passport", "паспорт", "inn", "инн", "snils", "снилс"}},
	{category: "address", patterns: []string{"address", "addr", "location", "адрес"}},
	{category: "birth_date", patterns: []string{"birth_date", "date_of_birth", "dob", "birthday", "датарождения", "датарожден"}}, // normalized
}

func detectSensitiveFields(rows []appports.ReportRow) []SensitiveField {
	fields := make([]SensitiveField, 0)

	for _, row := range rows {
		category, reason, ok := matchSensitiveField(row.ColumnName, derefString(row.ColumnComment))
		if !ok {
			continue
		}

		fields = append(fields, SensitiveField{
			SourceName:  row.SourceName,
			DatasetName: row.DatasetName,
			ColumnName:  row.ColumnName,
			Category:    category,
			Reason:      reason,
		})
	}

	return fields
}

func matchSensitiveField(columnName, columnComment string) (string, string, bool) {
	normalizedName := normalizeSensitiveText(columnName)
	normalizedComment := normalizeSensitiveText(columnComment)

	for _, rule := range sensitiveRules {
		for _, pattern := range rule.patterns {
			if normalizedName != "" && strings.Contains(normalizedName, pattern) {
				return rule.category, fmt.Sprintf("matched column name pattern %q", pattern), true
			}
			if normalizedComment != "" && strings.Contains(normalizedComment, pattern) {
				return rule.category, fmt.Sprintf("matched column comment pattern %q", pattern), true
			}
		}
	}

	return "", "", false
}

func normalizeSensitiveText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}

	replacer := strings.NewReplacer(
		"-", "_",
		" ", "_",
		".", "_",
		"/", "_",
		"(", "",
		")", "",
	)

	return replacer.Replace(value)
}
