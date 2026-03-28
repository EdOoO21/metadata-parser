package cli

import "testing"

func TestHashBytes(t *testing.T) {
	t.Parallel()

	first := hashBytes([]byte("abc"))
	second := hashBytes([]byte("abc"))
	third := hashBytes([]byte("abcd"))

	if first != second {
		t.Fatalf("expected deterministic hash, got %q and %q", first, second)
	}
	if first == third {
		t.Fatalf("expected different hashes, got %q and %q", first, third)
	}
	if len(first) != 64 {
		t.Fatalf("expected sha256 hex length 64, got %d", len(first))
	}
}
