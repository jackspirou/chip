package main

import (
	"bytes"
	"strings"
	"testing"
)

// `chip run test/gcd_main.chp` streams the program and prints 21.
func TestRunGCDFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"run", "../../examples/gcd.chp"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("run: %v (stderr: %s)", err, stderr.String())
	}
	if got := stdout.String(); got != "21\n" {
		t.Fatalf("stdout = %q, want %q", got, "21\n")
	}
}

// The bare-file shorthand behaves like `chip run`.
func TestRunShorthand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"../../examples/gcd.chp"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("run: %v (stderr: %s)", err, stderr.String())
	}
	if got := stdout.String(); got != "21\n" {
		t.Fatalf("stdout = %q, want %q", got, "21\n")
	}
}

// `chip repl` carries definitions across lines: f is defined on one line and
// called on the next.
func TestReplPersistsState(t *testing.T) {
	in := strings.NewReader("func f() int { return 42 }\nprint(f())\n")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"repl"}, in, &stdout, &stderr); err != nil {
		t.Fatalf("run repl: %v", err)
	}
	if got := stdout.String(); got != "42\n" {
		t.Fatalf("repl stdout = %q, want %q", got, "42\n")
	}
}
