package main

import (
	"bytes"
	"os"
	"path/filepath"
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

// `chip fmt --check` fails on unformatted source and passes once it is
// formatted.
func TestFmtCheck(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.chp")
	if err := os.WriteFile(path, []byte("func  main( ){print( 1 )}"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if err := run([]string{"fmt", "--check", path}, nil, &out, &errOut); err == nil {
		t.Fatal("expected --check to fail on unformatted source")
	}

	out.Reset()
	errOut.Reset()
	if err := run([]string{"fmt", "-w", path}, nil, &out, &errOut); err != nil {
		t.Fatalf("fmt -w: %v", err)
	}

	out.Reset()
	errOut.Reset()
	if err := run([]string{"fmt", "--check", path}, nil, &out, &errOut); err != nil {
		t.Fatalf("expected --check to pass after formatting, got %v (%s)", err, errOut.String())
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
