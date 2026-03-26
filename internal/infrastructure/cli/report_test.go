package cli

import "testing"

func TestValidateReportSelection(t *testing.T) {
	t.Parallel()

	if err := validateReportSelection(false, 42); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := validateReportSelection(false, 0); err == nil {
		t.Fatal("expected error for missing --run-id")
	}
}

func TestValidateReportSelection_PanicsOnLatest(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for latest selector")
		}
	}()

	_ = validateReportSelection(true, 0)
}
