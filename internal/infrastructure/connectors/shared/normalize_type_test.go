package shared

import (
	"testing"

	"github.com/EdOoO21/metadata-parser/internal/domain/types"
)

func TestNormalizeType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  types.CanonicalType
	}{
		{name: "empty", input: "", want: types.CanonicalTypeString},
		{name: "bool", input: "boolean", want: types.CanonicalTypeBoolean},
		{name: "number", input: "numeric(10,2)", want: types.CanonicalTypeNumber},
		{name: "timestamp", input: "timestamp with time zone", want: types.CanonicalTypeTimestamp},
		{name: "array", input: "text[]", want: types.CanonicalTypeArray},
		{name: "fallback", input: "varchar", want: types.CanonicalTypeString},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeType(tt.input); got != tt.want {
				t.Fatalf("NormalizeType(%q) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}
