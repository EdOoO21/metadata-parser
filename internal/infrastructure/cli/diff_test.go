package cli

import "testing"

func TestValidateDiffSelection(t *testing.T) {
	t.Parallel()

	if err := validateDiffSelection(false, 41, 42); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := validateDiffSelection(false, 0, 42); err == nil {
		t.Fatal("expected error for missing run ids")
	}
}

func TestValidateDiffSelection_PanicsOnLatest(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for latest selector")
		}
	}()

	_ = validateDiffSelection(true, 0, 0)
}
