package logging

import "testing"

func TestNewLoggerAndMethods(t *testing.T) {
	t.Parallel()

	logger := NewLogger()
	if logger == nil {
		t.Fatal("expected logger")
	}
	if logger.base == nil {
		t.Fatal("expected underlying slog logger")
	}

	logger.Info("info message", "key", "value")
	logger.Warn("warn message", "key", "value")
	logger.Error("error message", "key", "value")
}
