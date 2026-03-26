package report

import (
	"context"
	"testing"
)

func TestReportCatalogUseCase_ReturnsPlaceholder(t *testing.T) {
	t.Parallel()

	uc := NewReportCatalogUseCase()

	got, err := uc.Execute(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "report generation is not implemented yet: run_id=42"
	if got != want {
		t.Fatalf("unexpected message: got %q want %q", got, want)
	}
}
