package cli

import "testing"

func TestValidateDiffSelection(t *testing.T) {
	t.Parallel()

	if err := validateDiffSelection(false, 41, 42); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateDiffSelection(true, 0, 0); err != nil {
		t.Fatalf("unexpected error for latest selector: %v", err)
	}

	if err := validateDiffSelection(false, 0, 42); err == nil {
		t.Fatal("expected error for missing run ids")
	}
	if err := validateDiffSelection(true, 41, 42); err == nil {
		t.Fatal("expected error for conflicting latest and explicit run ids")
	}
}
