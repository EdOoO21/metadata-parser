package sensitive

import (
	"fmt"
	"strings"
)

type FieldMatch struct {
	Category string
	Reason   string
}

type rule struct {
	category string
	patterns []string
}

var rules = []rule{
	{category: "person_name", patterns: []string{"fio", "фио", "full_name", "fullname", "first_name", "last_name", "middle_name", "surname", "family_name", "given_name"}},
	{category: "contact", patterns: []string{"email", "e_mail", "mail", "phone", "mobile", "telephone", "tel"}},
	{category: "government_id", patterns: []string{"passport", "паспорт", "inn", "инн", "snils", "снилс"}},
	{category: "address", patterns: []string{"address", "addr", "location", "адрес"}},
	{category: "birth_date", patterns: []string{"birth_date", "date_of_birth", "dob", "birthday", "датарождения", "датарожден"}},
}

func MatchField(columnName, columnComment string) (FieldMatch, bool) {
	normalizedName := normalizeText(columnName)
	normalizedComment := normalizeText(columnComment)

	for _, rule := range rules {
		for _, pattern := range rule.patterns {
			if normalizedName != "" && strings.Contains(normalizedName, pattern) {
				return FieldMatch{
					Category: rule.category,
					Reason:   fmt.Sprintf("matched column name pattern %q", pattern),
				}, true
			}
			if normalizedComment != "" && strings.Contains(normalizedComment, pattern) {
				return FieldMatch{
					Category: rule.category,
					Reason:   fmt.Sprintf("matched column comment pattern %q", pattern),
				}, true
			}
		}
	}

	return FieldMatch{}, false
}

func normalizeText(value string) string {
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
