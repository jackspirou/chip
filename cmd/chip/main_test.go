package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackspirou/chip/internal/diag"
	"github.com/jackspirou/chip/internal/fix"
	"github.com/jackspirou/chip/internal/stream"
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

// `chip run` executes the multi-file packages example: an entry program that
// imports a local two-file geometry package (resolved relative to the file) and
// the built-in math package, printing 12 then 21 (Slice 6).
func TestRunPackagesExample(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"run", "../../examples/packages/main.chp"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("run: %v (stderr: %s)", err, stderr.String())
	}
	if got := stdout.String(); got != "12\n21\n" {
		t.Fatalf("stdout = %q, want %q", got, "12\n21\n")
	}
}

// `chip fmt -l` lists (and fails on, exit 2) unformatted source and passes once
// it is formatted — the exit bit is the signal (D16).
func TestFmtList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.chp")
	if err := os.WriteFile(path, []byte("func  main( ){print( 1 )}"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if err := run([]string{"fmt", "-l", path}, nil, &out, &errOut); err == nil {
		t.Fatal("expected -l to fail on unformatted source")
	} else if code := exitStatus(err); code != 2 {
		t.Fatalf("fmt -l on non-canonical: exit %d, want 2", code)
	}
	if got := strings.TrimSpace(out.String()); got != path {
		t.Fatalf("fmt -l listed %q, want the path %q", got, path)
	}

	out.Reset()
	errOut.Reset()
	if err := run([]string{"fmt", "-w", path}, nil, &out, &errOut); err != nil {
		t.Fatalf("fmt -w: %v", err)
	}

	out.Reset()
	errOut.Reset()
	if err := run([]string{"fmt", "-l", path}, nil, &out, &errOut); err != nil {
		t.Fatalf("expected -l to pass after formatting, got %v (%s)", err, errOut.String())
	}
	if out.Len() != 0 {
		t.Fatalf("fmt -l on canonical source listed %q, want nothing", out.String())
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

	// Already canonical: -l passes.
	out.Reset()
	errOut.Reset()
	if err := run([]string{"fmt", "-l", path}, nil, &out, &errOut); err != nil {
		t.Fatalf("expected -l to pass after formatting, got %v (%s)", err, errOut.String())
	}
}

// `chip lint` works on a program that imports and uses a package: the qualified
// call no longer trips the batch checker, so a clean program lints clean.
func TestLintWorksWithImports(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.chp")
	if err := os.WriteFile(path,
		[]byte("import \"geometry\"\nfunc main() { print(geometry.Area(3, 4)) }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if err := run([]string{"lint", path}, nil, &out, &errOut); err != nil {
		t.Fatalf("lint on a program with imports should succeed, got %v (%s)", err, errOut.String())
	}
	if errOut.Len() != 0 {
		t.Fatalf("expected no lint output, got %q", errOut.String())
	}
}

// `chip dump` reports a clean, documented boundary on a program with imports
// (the bytecode compiler does not support packages) — never a panic.
func TestDumpReportsUnsupportedImports(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.chp")
	if err := os.WriteFile(path,
		[]byte("import \"geometry\"\nfunc main() { print(geometry.Area(3, 4)) }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if err := run([]string{"dump", path}, nil, &out, &errOut); err == nil {
		t.Fatal("expected dump to report packages as unsupported")
	}
	if !strings.Contains(errOut.String(), "does not support packages") {
		t.Fatalf("stderr = %q, want a clean unsupported-packages message", errOut.String())
	}
}

// `chip dump` still disassembles an ordinary (import-free) program.
func TestDumpDisassemblesWithoutImports(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run([]string{"dump", "../../examples/gcd.chp"}, nil, &out, &errOut); err != nil {
		t.Fatalf("dump: %v (%s)", err, errOut.String())
	}
	if !strings.Contains(out.String(), "gcd") {
		t.Fatalf("expected disassembly of gcd, got %q", out.String())
	}
}

// `chip ast` prints the syntax tree of a program with imports, including the
// import spec and the qualified selector — no checker or compiler involved.
func TestAstPrintsImports(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.chp")
	if err := os.WriteFile(path,
		[]byte("import \"geometry\"\nfunc main() { print(geometry.Area(3, 4)) }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if err := run([]string{"ast", path}, nil, &out, &errOut); err != nil {
		t.Fatalf("ast: %v (%s)", err, errOut.String())
	}
	got := out.String()
	if !strings.Contains(got, "(import \"geometry\")") {
		t.Fatalf("ast output missing import spec:\n%s", got)
	}
	if !strings.Contains(got, "(sel geometry Area)") {
		t.Fatalf("ast output missing qualified selector:\n%s", got)
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

// A1: `printf 'print(6*7)\n' | chip run -` streams stdin and prints 42.
func TestRunStdinDash(t *testing.T) {
	in := strings.NewReader("print(6*7)\n")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"run", "-"}, in, &stdout, &stderr); err != nil {
		t.Fatalf("run -: %v (stderr: %s)", err, stderr.String())
	}
	if got := stdout.String(); got != "42\n" {
		t.Fatalf("stdout = %q, want %q", got, "42\n")
	}
}

// `chip run` with no arg reads a piped (non-terminal) stdin.
func TestRunStdinNoArg(t *testing.T) {
	in := strings.NewReader("print(1 + 2)\n")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"run"}, in, &stdout, &stderr); err != nil {
		t.Fatalf("run: %v (stderr: %s)", err, stderr.String())
	}
	if got := stdout.String(); got != "3\n" {
		t.Fatalf("stdout = %q, want %q", got, "3\n")
	}
}

// `chip run -e '<src>'` runs inline source.
func TestRunInlineE(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"run", "-e", "print(6 * 7)"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("run -e: %v (stderr: %s)", err, stderr.String())
	}
	if got := stdout.String(); got != "42\n" {
		t.Fatalf("stdout = %q, want %q", got, "42\n")
	}
}

// `-e` without a source argument is a usage error.
func TestRunInlineEMissingArg(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"run", "-e"}, nil, &stdout, &stderr); err == nil {
		t.Fatalf("run -e with no source: want error, got nil (stdout: %s)", stdout.String())
	}
}

// Other front-ends accept `-` for stdin too; fmt canonicalizes a piped program.
func TestFmtStdinDash(t *testing.T) {
	in := strings.NewReader("func  main(){print( 1 )}\n")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"fmt", "-"}, in, &stdout, &stderr); err != nil {
		t.Fatalf("fmt -: %v (stderr: %s)", err, stderr.String())
	}
	want := "func main() {\n    print(1)\n}\n"
	if got := stdout.String(); got != want {
		t.Fatalf("fmt - stdout = %q, want %q", got, want)
	}
}

// `chip ast -e` parses inline source.
func TestAstInlineE(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"ast", "-e", "print(7)"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("ast -e: %v (stderr: %s)", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "call print 7") {
		t.Fatalf("ast -e stdout = %q, want it to contain %q", stdout.String(), "call print 7")
	}
}

// A diagnostic from stdin names the source "<stdin>" and keeps the exact
// line:col it would report for a file (no position drift).
func TestStdinDiagnosticName(t *testing.T) {
	in := strings.NewReader("func main() { print(1 + ) }\n")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"run", "-"}, in, &stdout, &stderr); err == nil {
		t.Fatalf("want error from malformed stdin program, got nil")
	}
	if !strings.Contains(stderr.String(), "<stdin>:1:25:") {
		t.Fatalf("stderr = %q, want it to contain %q", stderr.String(), "<stdin>:1:25:")
	}
}

// `-w` has nowhere to write back stdin/-e source, so it is refused rather than
// creating a file literally named "<stdin>".
func TestFmtWriteStdinRefused(t *testing.T) {
	in := strings.NewReader("x := 1\n")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"fmt", "-w", "-"}, in, &stdout, &stderr); err == nil {
		t.Fatalf("fmt -w -: want error, got nil")
	}
	if _, err := os.Stat("<stdin>"); err == nil {
		os.Remove("<stdin>")
		t.Fatalf("fmt -w - created a file named <stdin>")
	}
}

// 3.4 (D22): the process exit code is the machine-readable signal a harness
// branches on without scraping text. Each row maps a kind of outcome to its
// sysexits-aligned code — 0 clean, 1 ran-then-threw, 2 static error
// (parse/type), 64 usage (bad flag/format/missing -e arg), 66 no input.
func TestExitCodeTable(t *testing.T) {
	// exitStatus prints usage/no-input errors to os.Stderr (there is no source
	// to render); silence it so the error rows don't spew "chip: …" into the log.
	if devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0); err == nil {
		saved := os.Stderr
		os.Stderr = devnull
		defer func() { os.Stderr = saved; devnull.Close() }()
	}

	cases := []struct {
		name  string
		args  []string
		stdin string
		want  int
	}{
		{"success", []string{"run", "-e", "print(1)"}, "", 0},
		{"runtime fault", []string{"run", "-e", "print(1 / 0)"}, "", 1},
		{"parse error", []string{"run", "-e", "print(1 +"}, "", 2},
		{"type error", []string{"run", "-e", "print(undef)"}, "", 2},
		{"bad format", []string{"run", "--format=bogus", "-e", "print(1)"}, "", 64},
		{"unknown flag", []string{"fmt", "--bogus", "-"}, "x := 1\n", 64},
		{"missing file", []string{"run", "/no/such/chip-file.chp"}, "", 66},
		{"-e without arg", []string{"run", "-e"}, "", 64},
		// Phase 4.1 caps. A4: a runaway loop trips the step budget (125). A flag
		// after the source is still parsed (caps scan every position).
		{"max-steps cap", []string{"run", "-", "--max-steps=100000"}, "func run() { i := 0\nfor i < 100000000 { i = i + 1 } }\nrun()\n", 125},
		{"max-output cap", []string{"run", "-", "--max-output=5"}, "func main() { i := 0\nfor i < 100 { print(\"x\")\ni = i + 1 } }\n", 125},
		{"timeout cap", []string{"run", "--timeout=1ms", "-"}, "func main() { for { } }\n", 124},
		{"bad cap value", []string{"run", "-", "--max-steps=nope"}, "print(1)\n", 64},
		// Phase 4.2 sealing. --no-imports refuses a host import (unstructured load
		// error → 1, like any failed import); --no-prelude drops abs (undefined →
		// static 2). Both flags scan every position, like the caps.
		{"no-imports refused", []string{"run", "--no-imports", "-"}, "import \"greet\"\nprint(1)\n", 1},
		{"no-prelude undefined", []string{"run", "-", "--no-prelude"}, "print(abs(0 - 1))\n", 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := run(c.args, strings.NewReader(c.stdin), &stdout, &stderr)
			if got := exitStatus(err); got != c.want {
				t.Fatalf("exit = %d, want %d (stderr: %s)", got, c.want, stderr.String())
			}
		})
	}
}

// exitCodeFor splits the resource phase by code: a wall-clock timeout is 124, a
// step or output budget is 125 — both distinct from a runtime fault (1). This is
// the mapping a harness branches on, kept honest independent of flag timing.
func TestExitCodeForResourceCaps(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"timeout", stream.ResourceError{Msg: "execution timed out"}, 124},
		{"steps", stream.ResourceError{Msg: "step budget exhausted (--max-steps=10)"}, 125},
		{"output", stream.ResourceError{Msg: "output limit reached (--max-output=10)"}, 125},
		{"runtime fault", stream.RuntimeError{Msg: "division by zero"}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := exitCodeFor(c.err); got != c.want {
				t.Fatalf("exitCodeFor(%v) = %d, want %d", c.err, got, c.want)
			}
		})
	}
}

// A tripped cap reports through the agent renderer like any other diagnostic:
// the code is greppable (error[RES002]) and the message names the flag, so a
// harness can route the fault and knows which knob to turn.
func TestRunStepCapReportsCode(t *testing.T) {
	const src = "func run() { i := 0\nfor i < 100000000 { i = i + 1 } }\nrun()\n"
	var stdout, stderr bytes.Buffer
	err := run([]string{"run", "-", "--max-steps=100000"}, strings.NewReader(src), &stdout, &stderr)
	var ee *exitError
	if !errors.As(err, &ee) || ee.code != 125 {
		t.Fatalf("err = %v, want *exitError with code 125", err)
	}
	out := stderr.String()
	if !strings.Contains(out, "error[RES002]") {
		t.Fatalf("stderr = %q, want it to carry error[RES002]", out)
	}
	if !strings.Contains(out, "--max-steps") {
		t.Fatalf("stderr = %q, want it to name --max-steps", out)
	}
}

// Plan §4.2 Verify: an import under --no-imports errors cleanly — a message that
// names the path and the reason, the program never runs (empty stdout), and the
// exit is 1 (a refused import is an unstructured load error, the same shape as any
// failed import). chip's only host-access vector is the loader, so sealing it
// makes the run hermetic.
func TestRunNoImportsRefusesCleanly(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"run", "--no-imports", "-"}, strings.NewReader("import \"greet\"\nprint(1)\n"), &stdout, &stderr)
	if code := exitStatus(err); code != 1 {
		t.Fatalf("exit = %d, want 1 (stderr: %s)", code, stderr.String())
	}
	if got := stderr.String(); !strings.Contains(got, "imports are disabled") || !strings.Contains(got, "greet") {
		t.Fatalf("stderr = %q, want it to name the refused import and the reason", got)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty (the program never ran)", stdout.String())
	}
}

// --no-prelude is inert for a program that only uses builtins: print and len are
// built into the engine, not the prelude, so a hermetic run still produces output.
func TestRunNoPreludeKeepsBuiltins(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"run", "--no-prelude", "-"}, strings.NewReader("print(len(\"hi\"))\n"), &stdout, &stderr); err != nil {
		t.Fatalf("unexpected error: %v (stderr: %s)", err, stderr.String())
	}
	if stdout.String() != "2\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "2\n")
	}
}

// A5 (plan §4.3): --save-on-error captures the *whole* program after a fault,
// including the lines past the error. Execution halts at the undefined name on
// line 1, but the source is buffered up front, so the saved file still carries
// all three lines — the capture a harness re-feeds or shows the user.
func TestRunSaveOnErrorCapturesPastFault(t *testing.T) {
	const prog = "print(oops)\nprint(2)\nprint(3)\n"
	path := filepath.Join(t.TempDir(), "p.chp") // does not exist yet
	var stdout, stderr bytes.Buffer
	if err := run([]string{"run", "-", "--save-on-error=" + path}, strings.NewReader(prog), &stdout, &stderr); err == nil {
		t.Fatal("want an error from the undefined name oops")
	}
	got, rerr := os.ReadFile(path)
	if rerr != nil {
		t.Fatalf("--save-on-error did not write the file: %v", rerr)
	}
	if string(got) != prog {
		t.Fatalf("saved source = %q, want every line %q", got, prog)
	}
}

// --save-on-error writes nothing when the run is clean: the capture is for faults
// only, so a successful program leaves no file behind.
func TestRunSaveOnErrorSkipsOnSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.chp")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"run", "-", "--save-on-error=" + path}, strings.NewReader("print(1)\n"), &stdout, &stderr); err != nil {
		t.Fatalf("run: %v (stderr: %s)", err, stderr.String())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("--save-on-error wrote a file for a clean run (stat err: %v)", err)
	}
}

// --save captures after a *successful* run too (unlike --save-on-error): the file
// holds the exact source that ran.
func TestRunSaveAlwaysCapturesOnSuccess(t *testing.T) {
	const prog = "print(6 * 7)\n"
	path := filepath.Join(t.TempDir(), "p.chp")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"run", "-", "--save=" + path}, strings.NewReader(prog), &stdout, &stderr); err != nil {
		t.Fatalf("run --save: %v (stderr: %s)", err, stderr.String())
	}
	if stdout.String() != "42\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "42\n")
	}
	if got, _ := os.ReadFile(path); string(got) != prog {
		t.Fatalf("saved source = %q, want %q", got, prog)
	}
}

// A6 (plan §4.3): an existing --save target is not clobbered. Without --force the
// run is refused up front (exit 73, EX_CANTCREAT) and never executes, leaving the
// file untouched; --force opts into overwriting it with the source that ran.
func TestRunSaveRefusesClobber(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.chp")
	const existing = "// do not clobber me\n"
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err := run([]string{"run", "-", "--save=" + path}, strings.NewReader("print(1)\n"), &stdout, &stderr)
	if code := exitStatus(err); code != 73 {
		t.Fatalf("exit = %d, want 73 (stderr: %s)", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty (the program must not run on a refused clobber)", stdout.String())
	}
	if got, _ := os.ReadFile(path); string(got) != existing {
		t.Fatalf("clobber refused but the file changed: %q", got)
	}

	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"run", "-", "--save=" + path, "--force"}, strings.NewReader("print(1)\n"), &stdout, &stderr); err != nil {
		t.Fatalf("run --save --force: %v (stderr: %s)", err, stderr.String())
	}
	if stdout.String() != "1\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "1\n")
	}
	if got, _ := os.ReadFile(path); string(got) != "print(1)\n" {
		t.Fatalf("--force did not overwrite with the source: %q", got)
	}
}

// A bare --save (no path) captures to a fresh, non-clobbering name in $TMPDIR and
// echoes where the source landed — so a caller that did not pick a path can still
// find it.
func TestRunSaveDefaultsToTmpdir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir) // os.CreateTemp("", …) writes under os.TempDir()
	const prog = "print(1)\n"
	var stdout, stderr bytes.Buffer
	if err := run([]string{"run", "-", "--save"}, strings.NewReader(prog), &stdout, &stderr); err != nil {
		t.Fatalf("run --save: %v (stderr: %s)", err, stderr.String())
	}
	entries, err := filepath.Glob(filepath.Join(dir, "chip-*.chp"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("want exactly one chip-*.chp in $TMPDIR, got %v (err %v)", entries, err)
	}
	if got, _ := os.ReadFile(entries[0]); string(got) != prog {
		t.Fatalf("saved source = %q, want %q", got, prog)
	}
	if !strings.Contains(stderr.String(), entries[0]) {
		t.Fatalf("stderr should echo the saved path %q, got %q", entries[0], stderr.String())
	}
}

// A8: off a tty (a *bytes.Buffer is not a *os.File) `chip run` renders the agent
// format — every diagnostic line is greppable as "error[CODE]:" and no ANSI
// color leaks in, so a harness can count and route faults by code.
func TestRunAgentFormatGreppable(t *testing.T) {
	in := strings.NewReader("print(undef)\n")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"run", "-"}, in, &stdout, &stderr); err == nil {
		t.Fatal("want an error from an undefined name")
	}
	out := stderr.String()
	if strings.Contains(out, "\x1b") {
		t.Fatalf("agent output should carry no ANSI color: %q", out)
	}
	coded := 0
	for line := range strings.SplitSeq(strings.TrimRight(out, "\n"), "\n") {
		if strings.HasPrefix(line, "error[") {
			coded++
		}
	}
	if coded == 0 {
		t.Fatalf("want at least one greppable error[CODE]: line, got:\n%s", out)
	}
}

// --format=llm renders the grounded LLM view: the same greppable error[CODE]:
// lines as agent, then a "how to fix:" appendix carrying the catalog's repair
// guidance for each code present (§5.5). Both the bare "llm" spec and the
// "llm:<model>" per-model form resolve to it.
func TestRunLLMFormatGrounded(t *testing.T) {
	entry, ok := diag.Explain("NAM001")
	if !ok {
		t.Fatal("catalog has no NAM001 entry")
	}
	for _, format := range []string{"--format=llm", "--format=llm:claude-opus-4-8"} {
		var stdout, stderr bytes.Buffer
		if err := run([]string{"run", format, "-e", "print(undef)"}, nil, &stdout, &stderr); err == nil {
			t.Fatalf("%s: want an error from an undefined name", format)
		}
		out := stderr.String()
		if !strings.Contains(out, "error[NAM001]") {
			t.Fatalf("%s: missing the greppable coded line:\n%s", format, out)
		}
		if !strings.Contains(out, "how to fix:") {
			t.Fatalf("%s: missing the grounded appendix:\n%s", format, out)
		}
		if !strings.Contains(out, entry.Fix) {
			t.Fatalf("%s: missing NAM001 fix guidance %q:\n%s", format, entry.Fix, out)
		}
	}
}

// --format=human forces the caret view even off a tty, so a developer piping to
// a pager still gets the located message and the underlined source line.
func TestRunHumanFormatCaret(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"run", "--format=human", "-e", "print(undef)"}, nil, &stdout, &stderr); err == nil {
		t.Fatal("want an error from an undefined name")
	}
	out := stderr.String()
	if !strings.Contains(out, "<arg>:1:7: undefined: undef") {
		t.Fatalf("human output missing the located message: %q", out)
	}
	if !strings.Contains(out, "\n  print(undef)\n") || !strings.Contains(out, "^") {
		t.Fatalf("human output missing the caret block: %q", out)
	}
}

// --json emits the Result envelope on stderr: ok:false carrying the diagnostics
// a consumer parses instead of scraping text. Program output (stdout) stays
// clean (D7).
func TestRunJSONFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"run", "--json", "-e", "print(undef)"}, nil, &stdout, &stderr); err == nil {
		t.Fatal("want an error from an undefined name")
	}
	if stdout.Len() != 0 {
		t.Fatalf("diagnostics must not leak onto stdout: %q", stdout.String())
	}
	var res diag.Result
	if err := json.Unmarshal(stderr.Bytes(), &res); err != nil {
		t.Fatalf("stderr is not the JSON envelope: %v\n%s", err, stderr.String())
	}
	if res.OK {
		t.Fatal("ok should be false when a diagnostic is present")
	}
	if len(res.Diagnostics) == 0 {
		t.Fatal("want at least one diagnostic in the envelope")
	}
}

// `chip fmt -d` writes the unified diff that -w would apply and exits 2, leaving
// the file untouched — a hook can show the reformatting without rewriting.
func TestFmtDiff(t *testing.T) {
	const orig = "func  main( ){print( 1 )}"
	path := filepath.Join(t.TempDir(), "p.chp")
	if err := os.WriteFile(path, []byte(orig), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	err := run([]string{"fmt", "-d", path}, nil, &out, &errOut)
	if code := exitStatus(err); code != 2 {
		t.Fatalf("fmt -d on non-canonical: exit %d, want 2 (stderr: %s)", code, errOut.String())
	}
	diff := out.String()
	if !strings.Contains(diff, "@@ ") {
		t.Fatalf("diff missing a hunk header: %q", diff)
	}
	if !strings.Contains(diff, "+func main() {") {
		t.Fatalf("diff missing the reformatted line: %q", diff)
	}
	if got, _ := os.ReadFile(path); string(got) != orig {
		t.Fatalf("fmt -d modified the file: %q", got)
	}
}

// §5.2: `chip prompt` writes the agent-facing contract to stdout — the stream and
// validate commands, the exit-code table, the JSON Result shape, the code catalog,
// and the resource caps — so a harness can curl the contract straight into context.
// Each element is asserted so the contract can never silently lose a part.
func TestPromptEmitsContract(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"prompt"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("prompt: %v (stderr: %s)", err, stderr.String())
	}
	got := stdout.String()
	for _, want := range []string{
		"chip run -",         // stream a program in
		"chip check -",       // validate everything at once
		"| code | meaning |", // the exit-code table
		"Result",             // the JSON envelope shape
		`"ok"`,               // a field of that shape
		"PAR001",             // the code catalog…
		"RES003",             // …including its resource codes
		"--max-steps",        // the resource caps
		"--timeout",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("prompt output is missing %q:\n%s", want, got)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("prompt writes only to stdout, got stderr: %q", stderr.String())
	}
}

// The contract `chip prompt` prints is //go:embed'd from prompt.md, and the
// bundled SKILL.md carries that same body verbatim as its tail (after the
// frontmatter). Asserting the suffix keeps the shipped skill and the binary's
// contract from drifting apart — the single-source-of-truth guarantee of §5.2.
func TestPromptEmbeddedInSkill(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"prompt"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("prompt: %v (stderr: %s)", err, stderr.String())
	}
	prompt := stdout.String()
	for _, skill := range []string{
		"../../.agents/skills/chip/SKILL.md",
		"../../.claude/skills/chip/SKILL.md",
	} {
		b, err := os.ReadFile(skill)
		if err != nil {
			t.Fatalf("read %s: %v", skill, err)
		}
		body := string(b)
		if !strings.HasSuffix(body, prompt) {
			t.Fatalf("%s does not embed `chip prompt` output verbatim as its tail", skill)
		}
		if !strings.HasPrefix(body, "---\nname: chip\n") {
			t.Fatalf("%s is missing the skill frontmatter", skill)
		}
	}
}

// `chip prompt` takes no arguments; a stray one is a usage error (exit 64), not a
// silently ignored flag — so a caller expecting, say, JSON is told plainly.
func TestPromptRejectsArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"prompt", "--json"}, nil, &stdout, &stderr)
	var ue *usageError
	if !errors.As(err, &ue) {
		t.Fatalf("err = %v, want a *usageError (exit 64)", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty on a usage error", stdout.String())
	}
}

// deadCodeProg's only issue is a statement after the first return: the canonical
// Machine fix. f is defined before main so there is no use-before-def warning to
// muddy the post-fix --strict check.
const deadCodeProg = `package main
func f() int {
    return 1
    return 2
}
func main() { print(f()) }`

// typoProg's only fixable issue is a misspelled reference to an in-scope variable:
// a "did you mean" (Maybe) that fix must never apply on its own.
const typoProg = `package main
func main() {
    answer := 42
    print(anser)
}`

// `chip fix` applies the Machine dead-code removal and writes the repaired source
// to stdout, leaving the rest of the program byte-for-byte intact.
func TestFixAppliesUnreachableToStdout(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"fix", "-"}, strings.NewReader(deadCodeProg), &stdout, &stderr); err != nil {
		t.Fatalf("fix -: %v (stderr: %s)", err, stderr.String())
	}
	want := `package main
func f() int {
    return 1
}
func main() { print(f()) }`
	if stdout.String() != want {
		t.Fatalf("fixed source:\n%q\nwant:\n%q", stdout.String(), want)
	}
}

// A9: after `chip fix` applies the machine repair, the program passes the static
// check it failed before — verified end to end by piping fix's output into
// `check --strict` (which includes lint). Both steps run in process.
func TestFixAppliedSourcePassesStrictCheck(t *testing.T) {
	// Before: --strict flags the dead code (exit 2).
	var pre bytes.Buffer
	if err := run([]string{"check", "--strict", "-"}, strings.NewReader(deadCodeProg), &pre, &pre); err == nil {
		t.Fatal("expected check --strict to fail on the dead code")
	} else if code := exitStatus(err); code != 2 {
		t.Fatalf("pre-fix check --strict exit = %d, want 2", code)
	}

	// Fix, capturing the repaired source.
	var fixed, stderr bytes.Buffer
	if err := run([]string{"fix", "-"}, strings.NewReader(deadCodeProg), &fixed, &stderr); err != nil {
		t.Fatalf("fix -: %v", err)
	}

	// After: the repaired source is clean under --strict (exit 0).
	var post bytes.Buffer
	if err := run([]string{"check", "--strict", "-"}, bytes.NewReader(fixed.Bytes()), &post, &post); err != nil {
		t.Fatalf("post-fix check --strict: %v (%s)", err, post.String())
	}
}

// The safety guardrail: a "did you mean" (Maybe) is never applied automatically,
// so `chip fix` returns the source unchanged.
func TestFixLeavesMaybeUnchanged(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"fix", "-"}, strings.NewReader(typoProg), &stdout, &stderr); err != nil {
		t.Fatalf("fix -: %v", err)
	}
	if stdout.String() != typoProg {
		t.Fatalf("fix changed source despite only a Maybe fix:\n%q", stdout.String())
	}
}

// `chip fix --plan --json` changes nothing and emits the machine plan: the
// unapplied Maybe is listed with apply=false and its concrete edit, so a harness
// can decide to opt it in.
func TestFixPlanJSONListsUnapplied(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"fix", "--plan", "--json", "-"}, strings.NewReader(typoProg), &stdout, &stderr); err != nil {
		t.Fatalf("fix --plan --json: %v", err)
	}
	var plan fix.Plan
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatalf("stdout did not parse as fix.Plan: %v\n%s", err, stdout.String())
	}
	if plan.Applied != 0 || plan.Skipped != 1 {
		t.Fatalf("plan applied=%d skipped=%d, want 0/1", plan.Applied, plan.Skipped)
	}
	if len(plan.Fixes) != 1 {
		t.Fatalf("plan has %d fixes, want 1", len(plan.Fixes))
	}
	pf := plan.Fixes[0]
	if pf.Apply {
		t.Error("planned Maybe fix marked apply=true, want false")
	}
	if pf.Code != "NAM001" {
		t.Errorf("planned fix code = %q, want NAM001", pf.Code)
	}
	if len(pf.Edits) == 0 {
		t.Error("planned fix lists no edits to opt into")
	}
}

// `chip fix -w` rewrites the file in place and notes how many fixes it applied.
func TestFixWriteInPlace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.chp")
	if err := os.WriteFile(path, []byte(deadCodeProg), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"fix", "-w", path}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("fix -w: %v (stderr: %s)", err, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("fix -w wrote to stdout: %q, want nothing", stdout.String())
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "return 2") {
		t.Errorf("dead code still present after fix -w:\n%s", got)
	}
	if !strings.Contains(stderr.String(), "applied 1") {
		t.Errorf("stderr = %q, want an applied count", stderr.String())
	}
}

// `chip fix -w` on stdin has nowhere to write back, so it is a usage error rather
// than a silent no-op.
func TestFixWriteRejectsStdin(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"fix", "-w", "-"}, strings.NewReader(deadCodeProg), &stdout, &stderr)
	var ue *usageError
	if !errors.As(err, &ue) {
		t.Fatalf("err = %v, want a *usageError (exit 64)", err)
	}
}

// An unknown flag to fix is a usage error, not a silently ignored argument.
func TestFixUnknownFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"fix", "--bogus", "-"}, strings.NewReader(deadCodeProg), &stdout, &stderr)
	var ue *usageError
	if !errors.As(err, &ue) {
		t.Fatalf("err = %v, want a *usageError (exit 64)", err)
	}
}

// The binary assembles the skill (frontmatter + contract) from its embeds, and
// the shipped SKILL.md copies must equal those bytes exactly — that equality is
// what makes the binary the single source of truth (D24): if prompt.md or the
// frontmatter changes without regenerating the files, this fails.
func TestSkillContentMatchesBundled(t *testing.T) {
	want := skillContent()
	for _, path := range []string{
		"../../.agents/skills/chip/SKILL.md",
		"../../.claude/skills/chip/SKILL.md",
	} {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if string(b) != want {
			t.Fatalf("%s is not byte-identical to skillContent(); regenerate it", path)
		}
	}
}

// `chip skill print` writes the complete SKILL.md to stdout, nothing to stderr —
// the Zero model (nothing on disk is touched).
func TestSkillPrintToStdout(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"skill", "print"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("skill print: %v (stderr: %s)", err, stderr.String())
	}
	if stdout.String() != skillContent() {
		t.Fatal("skill print stdout is not the assembled skill content")
	}
	if !strings.HasPrefix(stdout.String(), "---\nname: chip\n") {
		t.Fatal("skill print is missing the frontmatter")
	}
	if stderr.Len() != 0 {
		t.Fatalf("skill print wrote to stderr: %q", stderr.String())
	}
}

// `chip skill install` into a fresh repo writes both byte-identical SKILL.md
// copies plus a managed AGENTS.md block, and a second run is a no-op (every target
// reports unchanged, exit 0, no bytes move) — the §5.8 Verify.
func TestSkillInstallWritesAndIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	if err := run([]string{"skill", "install"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("skill install: %v (stderr: %s)", err, stderr.String())
	}
	want := skillContent()
	for _, rel := range []string{".agents/skills/chip/SKILL.md", ".claude/skills/chip/SKILL.md"} {
		b, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if string(b) != want {
			t.Fatalf("%s is not byte-identical to the shipped skill", rel)
		}
	}
	agents, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if !strings.Contains(string(agents), agentsBlockBegin) || !strings.Contains(string(agents), agentsBlockEnd) {
		t.Fatalf("AGENTS.md is missing the managed block:\n%s", agents)
	}

	// Re-install: a no-op. Snapshot every file, run again, assert nothing moved and
	// the command reports only "unchanged".
	before := snapshotTree(t, dir)
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"skill", "install"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("skill install (re-run): %v (stderr: %s)", err, stderr.String())
	}
	if strings.Contains(stdout.String(), "created") || strings.Contains(stdout.String(), "updated") {
		t.Fatalf("re-install was not a no-op:\n%s", stdout.String())
	}
	if after := snapshotTree(t, dir); !mapsEqual(before, after) {
		t.Fatal("re-install changed files on disk; install is not idempotent")
	}
}

// A SKILL.md that differs from what the binary ships (a user edit, or a stale
// copy) is refused without --force (exit 73, file untouched); --force overwrites
// it back to the shipped bytes.
func TestSkillInstallRefusesClobber(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	if err := run([]string{"skill", "install"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("skill install: %v (stderr: %s)", err, stderr.String())
	}
	edited := filepath.Join(dir, ".agents/skills/chip/SKILL.md")
	const userMark = "\nUSER EDIT\n"
	if err := os.WriteFile(edited, []byte(skillContent()+userMark), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	err := run([]string{"skill", "install"}, nil, &stdout, &stderr)
	if code := exitStatus(err); code != 73 {
		t.Fatalf("exit = %d, want 73 on a refused clobber (stderr: %s)", code, stderr.String())
	}
	if b, _ := os.ReadFile(edited); !strings.HasSuffix(string(b), userMark) {
		t.Fatal("refused clobber but the user-edited file changed")
	}

	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"skill", "install", "--force"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("skill install --force: %v (stderr: %s)", err, stderr.String())
	}
	if b, _ := os.ReadFile(edited); string(b) != skillContent() {
		t.Fatal("--force did not restore the shipped skill bytes")
	}
}

// `--agent NAME` is a surgical install: only that one harness's SKILL.md is
// written, and the cross-harness AGENTS.md block is left out. An unknown name is a
// usage error.
func TestSkillInstallAgentSelector(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	if err := run([]string{"skill", "install", "--agent", "claude"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("skill install --agent claude: %v (stderr: %s)", err, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude/skills/chip/SKILL.md")); err != nil {
		t.Fatalf("--agent claude did not write the claude skill: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".agents/skills/chip/SKILL.md")); !os.IsNotExist(err) {
		t.Fatal("--agent claude also wrote the agents skill (should be surgical)")
	}
	if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatal("--agent claude also wrote AGENTS.md (should be surgical)")
	}

	var ue *usageError
	if err := run([]string{"skill", "install", "--agent", "bogus"}, nil, &stdout, &stderr); !errors.As(err, &ue) {
		t.Fatalf("err = %v, want a *usageError for an unknown --agent", err)
	}
}

// `--print` dry-runs the skill to stdout and touches nothing on disk.
func TestSkillInstallPrintDryRun(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	if err := run([]string{"skill", "install", "--print"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("skill install --print: %v (stderr: %s)", err, stderr.String())
	}
	if stdout.String() != skillContent() {
		t.Fatal("skill install --print did not write the skill to stdout")
	}
	if n := len(snapshotTree(t, dir)); n != 0 {
		t.Fatalf("skill install --print wrote %d files; want 0", n)
	}
}

// `chip skill list` reports the skill name and, per target, whether a current copy
// is installed: absent before install, current after.
func TestSkillListReportsState(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	if err := run([]string{"skill", "list"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("skill list: %v (stderr: %s)", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "skill "+skillName()) {
		t.Fatalf("skill list did not report the skill name:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "absent") {
		t.Fatalf("skill list should report targets absent before install:\n%s", stdout.String())
	}

	if err := run([]string{"skill", "install"}, nil, &bytes.Buffer{}, &stderr); err != nil {
		t.Fatalf("skill install: %v (stderr: %s)", err, stderr.String())
	}
	stdout.Reset()
	if err := run([]string{"skill", "list"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("skill list (post-install): %v", err)
	}
	if !strings.Contains(stdout.String(), "installed (current)") {
		t.Fatalf("skill list should report installed (current) after install:\n%s", stdout.String())
	}
}

// Installing into a repo that already has a hand-authored AGENTS.md preserves the
// user's content and appends the managed block; a later edit outside the markers
// survives a re-install (the block alone is chip's to refresh).
func TestSkillAgentsBlockPreservesUserContent(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	const userDoc = "# My Project\n\nHand-written notes the user owns.\n"
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(userDoc), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := run([]string{"skill", "install"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("skill install: %v (stderr: %s)", err, stderr.String())
	}
	got, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if !strings.Contains(string(got), "Hand-written notes the user owns.") {
		t.Fatalf("install clobbered the user's AGENTS.md content:\n%s", got)
	}
	if !strings.Contains(string(got), agentsBlockBegin) {
		t.Fatalf("install did not append the managed block:\n%s", got)
	}

	// A user edit outside the markers must survive a re-install, and the block must
	// register as unchanged.
	const tail = "\nMore notes below the block.\n"
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), append(got, tail...), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"skill", "install"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("skill install (re-run): %v (stderr: %s)", err, stderr.String())
	}
	got2, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if !strings.Contains(string(got2), "More notes below the block.") {
		t.Fatalf("re-install dropped a user edit outside the markers:\n%s", got2)
	}
}

// `chip skill` with no subcommand, or an unknown one, is a usage error.
func TestSkillSubcommandErrors(t *testing.T) {
	for _, args := range [][]string{{"skill"}, {"skill", "bogus"}, {"skill", "print", "--json"}} {
		var stdout, stderr bytes.Buffer
		var ue *usageError
		if err := run(args, nil, &stdout, &stderr); !errors.As(err, &ue) {
			t.Fatalf("run(%v) err = %v, want a *usageError", args, err)
		}
	}
}

// snapshotTree maps every regular file under root to its contents, for asserting a
// command left the tree untouched.
func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[rel] = string(b)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return out
}

// mapsEqual reports whether two file snapshots are identical.
func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
