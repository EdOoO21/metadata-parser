package cli

import "testing"

func TestValidateReportSelection(t *testing.T) {
	t.Parallel()

	if err := validateReportSelection(false, 42); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateReportSelection(true, 0); err != nil {
		t.Fatalf("unexpected error for latest selector: %v", err)
	}

	if err := validateReportSelection(false, 0); err == nil {
		t.Fatal("expected error for missing --run-id")
	}
	if err := validateReportSelection(true, 42); err == nil {
		t.Fatal("expected error for conflicting latest and run-id")
	}
}
