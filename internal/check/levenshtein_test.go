package check

import "testing"

func TestLevenshtein(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "abc", 0},
		{"a", "", 1},
		{"", "abc", 3},
		{"kitten", "sitting", 3},
		{"count", "coont", 1},
		{"flaw", "lawn", 2},
		{"café", "cafe", 1}, // rune-based: the accented é is a single edit, not two bytes
	}
	for _, tc := range cases {
		if got := levenshtein(tc.a, tc.b); got != tc.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
		if got := levenshtein(tc.b, tc.a); got != tc.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d (symmetry)", tc.b, tc.a, got, tc.want)
		}
	}
}

func TestNearestPicksClosestWithinBound(t *testing.T) {
	got, ok := nearest("coont", []string{"print", "count", "f"})
	if !ok || got != "count" {
		t.Fatalf("nearest = (%q, %v), want (\"count\", true)", got, ok)
	}
}

// Equidistant candidates resolve to the lexicographically smaller name so the
// suggestion is deterministic — a precondition for reproducible repair loops.
func TestNearestTieBreaksLexicographically(t *testing.T) {
	got, ok := nearest("ab", []string{"ad", "ac"})
	if !ok || got != "ac" {
		t.Fatalf("nearest = (%q, %v), want (\"ac\", true)", got, ok)
	}
}

// Nothing within the length-scaled bound means no hint: a grounded suggestion is
// offered only when one is close, never guessed (D14).
func TestNearestOmitsWhenNothingClose(t *testing.T) {
	if got, ok := nearest("undef", []string{"print", "len", "main"}); ok {
		t.Fatalf("nearest = (%q, true), want no suggestion", got)
	}
}

// The undefined name itself is never suggested as its own fix.
func TestNearestSkipsExactMatch(t *testing.T) {
	if got, ok := nearest("count", []string{"count"}); ok {
		t.Fatalf("nearest = (%q, true), want no suggestion (only candidate equals target)", got)
	}
}
