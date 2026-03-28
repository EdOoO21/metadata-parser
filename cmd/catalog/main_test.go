package main

import (
	"testing"
)

func TestNewRootCmd(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd()
	if cmd == nil {
		t.Fatal("expected root command")
	}
	if cmd.Use != "catalog" {
		t.Fatalf("unexpected use: %s", cmd.Use)
	}

	commands := cmd.Commands()
	if len(commands) < 3 {
		t.Fatalf("expected at least 3 subcommands, got %d", len(commands))
	}
}
