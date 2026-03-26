package logging

import (
	"log/slog"
	"os"
)

type Logger struct {
	base *slog.Logger
}

func NewLogger() *Logger {
	return &Logger{
		base: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})),
	}
}

func (l *Logger) Info(msg string, args ...any) {
	l.base.Info(msg, args...)
}

func (l *Logger) Warn(msg string, args ...any) {
	l.base.Warn(msg, args...)
}

func (l *Logger) Error(msg string, args ...any) {
	l.base.Error(msg, args...)
}
