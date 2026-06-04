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

// `chip run main.chp` resolves an import relative to the file's directory and
// runs the qualified call (PE1 over the real filesystem).
func TestRunImportsRelativeToFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "geometry.chp"),
		[]byte("package geometry\nfunc Area(w int, h int) int { return w * h }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(dir, "main.chp")
	if err := os.WriteFile(mainPath,
		[]byte("import \"geometry\"\nfunc main() { print(geometry.Area(3, 4)) }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := run([]string{"run", mainPath}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("run: %v (stderr: %s)", err, stderr.String())
	}
	if got := stdout.String(); got != "12\n" {
		t.Fatalf("stdout = %q, want %q", got, "12\n")
	}
}

// `chip run main.chp` loads a directory package (geometry/ with two files) from
// the real filesystem and runs the cross-file call (PE6 over the FS).
func TestRunMultiFileDirPackage(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "geometry")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "area.chp"),
		[]byte("package geometry\nfunc Area(w int, h int) int { return scale(w * h) }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "scale.chp"),
		[]byte("package geometry\nfunc scale(n int) int { return n }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(dir, "main.chp")
	if err := os.WriteFile(mainPath,
		[]byte("import \"geometry\"\nfunc main() { print(geometry.Area(3, 4)) }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := run([]string{"run", mainPath}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("run: %v (stderr: %s)", err, stderr.String())
	}
	if got := stdout.String(); got != "12\n" {
		t.Fatalf("stdout = %q, want %q", got, "12\n")
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

// `chip lint` reports an unused import (PE8) over a real file.
func TestLintReportsUnusedImport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.chp")
	if err := os.WriteFile(path, []byte("import \"geometry\"\nfunc main() { print(1) }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if err := run([]string{"lint", path}, nil, &out, &errOut); err == nil {
		t.Fatal("expected lint to report the unused import")
	}
	if !strings.Contains(errOut.String(), "imported and not used") {
		t.Fatalf("stderr = %q, want an unused-import hint", errOut.String())
	}
}

// `chip fmt` sorts and dedupes an import block and is idempotent on a real file.
func TestFmtSortsImportsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.chp")
	if err := os.WriteFile(path,
		[]byte("import (\n\"sort\"\n\"fmt\"\n\"fmt\"\n)\nfunc main() { print(1) }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if err := run([]string{"fmt", "-w", path}, nil, &out, &errOut); err != nil {
		t.Fatalf("fmt -w: %v (%s)", err, errOut.String())
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(got), "\"fmt\""); n != 1 {
		t.Fatalf("duplicate import not removed, %d copies of \"fmt\":\n%s", n, got)
	}
	if strings.Index(string(got), "\"fmt\"") > strings.Index(string(got), "\"sort\"") {
		t.Fatalf("imports not sorted (fmt should precede sort):\n%s", got)
	}

	// Already canonical: --check passes.
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
