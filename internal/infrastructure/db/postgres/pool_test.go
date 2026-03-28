package postgres

import (
	"context"
	"testing"
)

func TestNewPoolFromEnv_EmptyEnv(t *testing.T) {
	t.Parallel()

	_, err := NewPoolFromEnv(context.Background(), "THIS_ENV_SHOULD_NOT_EXIST_FOR_METADATA_PARSER_TEST")
	if err == nil {
		t.Fatal("expected empty env error")
	}
}
