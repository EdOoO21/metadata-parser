package shared

import (
	"strings"

	"github.com/EdOoO21/metadata-parser/internal/domain/types"
)

func NormalizeType(originalType string) types.CanonicalType {
	value := strings.ToLower(strings.TrimSpace(originalType))

	switch {
	case value == "":
		return types.CanonicalTypeString
	case strings.Contains(value, "bool"):
		return types.CanonicalTypeBoolean
	case strings.Contains(value, "int"),
		strings.Contains(value, "numeric"),
		strings.Contains(value, "decimal"),
		strings.Contains(value, "real"),
		strings.Contains(value, "double"),
		strings.Contains(value, "float"),
		strings.Contains(value, "number"):
		return types.CanonicalTypeNumber
	case strings.Contains(value, "timestamp"),
		strings.Contains(value, "date"),
		strings.Contains(value, "time"):
		return types.CanonicalTypeTimestamp
	case strings.Contains(value, "array"),
		strings.HasSuffix(value, "[]"):
		return types.CanonicalTypeArray
	default:
		return types.CanonicalTypeString
	}
}
