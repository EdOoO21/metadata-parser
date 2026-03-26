package diff

import (
	"context"
	"testing"
)

func TestDiffCatalogUseCase_ReturnsPlaceholder(t *testing.T) {
	t.Parallel()

	uc := NewDiffCatalogUseCase()

	got, err := uc.Execute(context.Background(), 41, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "diff generation is not implemented yet: from_run_id=41 to_run_id=42"
	if got != want {
		t.Fatalf("unexpected message: got %q want %q", got, want)
	}
}
