package stream

import "testing"

// A Session carries state across Feed calls: a function defined in one frame is
// callable in the next, and its output is returned per frame (not to a sink).
func TestSessionFeedSharesState(t *testing.T) {
	s, err := NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if out, err := s.Feed("func double(n int) int { return n + n }\n"); err != nil || out != "" {
		t.Fatalf("define frame: out=%q err=%v, want empty/no error", out, err)
	}
	out, err := s.Feed("print(double(21))\n")
	if err != nil {
		t.Fatalf("call frame: %v", err)
	}
	if out != "42\n" {
		t.Fatalf("output = %q, want %q", out, "42\n")
	}
}

// A top-level variable bound in one frame keeps its value in a later frame.
func TestSessionTopLevelVarPersists(t *testing.T) {
	s, _ := NewSession()
	if _, err := s.Feed("x := 21\n"); err != nil {
		t.Fatalf("bind frame: %v", err)
	}
	out, err := s.Feed("print(x * 2)\n")
	if err != nil {
		t.Fatalf("use frame: %v", err)
	}
	if out != "42\n" {
		t.Fatalf("output = %q, want %q", out, "42\n")
	}
}

// A faulting frame returns a structured error and does not corrupt later frames.
func TestSessionFaultIsRecoverable(t *testing.T) {
	s, _ := NewSession()
	if _, err := s.Feed("print(undef)\n"); err == nil {
		t.Fatal("want an error for an undefined name")
	}
	out, err := s.Feed("print(6 * 7)\n")
	if err != nil {
		t.Fatalf("recovery frame: %v", err)
	}
	if out != "42\n" {
		t.Fatalf("output = %q, want %q", out, "42\n")
	}
}
