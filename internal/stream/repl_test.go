package stream

import (
	"bytes"
	"strings"
	"testing"
)

// REPL carries definitions across separate inputs: f is defined on one line and
// called on the next, sharing the engine's persistent state.
func TestREPLPersistsState(t *testing.T) {
	in := strings.NewReader("func f() int { return 42 }\nprint(f())\n")
	var out bytes.Buffer
	if err := REPL(in, &out); err != nil {
		t.Fatalf("REPL: %v", err)
	}
	if got := out.String(); got != "42\n" {
		t.Fatalf("stdout = %q, want %q", got, "42\n")
	}
}

// A persistent engine sees definitions from an earlier feed in a later one.
func TestEngineFeedSharesState(t *testing.T) {
	var out bytes.Buffer
	e := newEngine(&out, DirLoader("."))

	if err := e.feed(strings.NewReader("func double(n int) int { return n + n }\n")); err != nil {
		t.Fatalf("feed 1: %v", err)
	}
	if err := e.feed(strings.NewReader("print(double(21))\n")); err != nil {
		t.Fatalf("feed 2: %v", err)
	}
	if got := out.String(); got != "42\n" {
		t.Fatalf("stdout = %q, want %q", got, "42\n")
	}
}
