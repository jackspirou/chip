package compiler_test

import (
	"strings"
	"testing"

	"github.com/jackspirou/chip/internal/check"
	"github.com/jackspirou/chip/internal/code"
	"github.com/jackspirou/chip/internal/compiler"
	"github.com/jackspirou/chip/internal/parser"
)

// The compiler had no direct tests; it was exercised only through the VM tests.
// These pin its contracts: the structure of the program it emits, its refusal of
// the one thing it cannot compile (imports), and the backstops that keep
// chip.Compile() safe even when handed a tree a lenient checker let through.

// compile parses, type-checks (fatal on a check error — these tests are about the
// compiler, not the checker), and compiles src, returning the program and any
// compile error.
func compile(t *testing.T, src string) (*code.Program, error) {
	t.Helper()
	p, err := parser.New(strings.NewReader(src))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	f, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	info, err := check.Check(f)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return compiler.Compile(f, info)
}

// compileIgnoringCheck compiles src while discarding the checker's verdict, so a
// program the checker rejects still reaches the compiler. check.Check populates
// Info even when it returns an error, so the compiler has the symbol data it
// needs. This stands in for an embedding host that checks more leniently than
// chip, and exercises the compiler's own guards.
func compileIgnoringCheck(t *testing.T, src string) (*code.Program, error) {
	t.Helper()
	p, err := parser.New(strings.NewReader(src))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	f, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	info, _ := check.Check(f) // ignore check errors on purpose; Info is still populated
	return compiler.Compile(f, info)
}

// TestCompileProgramStructure checks the shape of the emitted program: one
// FuncProto per declared function in source order, the entry index pointing at
// main, and the parameter and local-slot counts the VM allocates frames from.
func TestCompileProgramStructure(t *testing.T) {
	prog, err := compile(t, `package main
func add(x int, y int) int { return x + y }
func main() {
    z := add(2, 3)
    print(z)
}`)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(prog.Functions) != 2 {
		t.Fatalf("Functions = %d, want 2", len(prog.Functions))
	}
	// Source order: add is declared first (index 0), main second (index 1).
	add, main := prog.Functions[0], prog.Functions[1]
	if add.Name != "add" || main.Name != "main" {
		t.Fatalf("function order = %q,%q, want add,main", add.Name, main.Name)
	}
	if prog.Entry != 1 {
		t.Errorf("Entry = %d, want 1 (main)", prog.Entry)
	}
	if add.NumParams != 2 || add.NumLocals != 2 {
		t.Errorf("add params/locals = %d/%d, want 2/2", add.NumParams, add.NumLocals)
	}
	// main has no params and one local (z).
	if main.NumParams != 0 || main.NumLocals != 1 {
		t.Errorf("main params/locals = %d/%d, want 0/1", main.NumParams, main.NumLocals)
	}
}

// TestCompileNoMainEntry checks that a file without a main compiles to a program
// whose Entry is -1, the sentinel the VM uses to refuse to run.
func TestCompileNoMainEntry(t *testing.T) {
	prog, err := compile(t, `package main
func helper() int { return 1 }`)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if prog.Entry != -1 {
		t.Errorf("Entry = %d, want -1 (no main)", prog.Entry)
	}
}

// TestCompileRefusesImports locks the documented boundary: the bytecode compiler
// does not support cross-package calls, so it refuses a file that imports rather
// than emit a half-working program. (The file type-checks clean — the batch
// checker treats imports opaquely — so this error comes from the compiler.)
func TestCompileRefusesImports(t *testing.T) {
	_, err := compile(t, `package main
import "geometry"
func main() { print(geometry.Area(3, 4)) }`)
	if err == nil {
		t.Fatal("compiling a file with imports succeeded, want refusal")
	}
	if !strings.Contains(err.Error(), "does not support packages") {
		t.Errorf("error = %q, want it to mention packages are unsupported", err.Error())
	}
}

// TestCompilerBackstopsFunctionAsValue is a defense-in-depth check. The batch
// checker now rejects a bare function used as a value, so chip.Compile() never
// reaches the compiler with one. But the compiler must still refuse it on its own
// — an embedding host may check more leniently — rather than emit a load of a
// non-existent local slot. With the check error ignored, the compiler reports
// that the function name is not a variable.
func TestCompilerBackstopsFunctionAsValue(t *testing.T) {
	_, err := compileIgnoringCheck(t, `package main
func f() int { return 1 }
func main() { print(f) }`)
	if err == nil {
		t.Fatal("compiler accepted a function used as a value, want an error")
	}
	if !strings.Contains(err.Error(), "f is not a variable") {
		t.Errorf("error = %q, want %q", err.Error(), "f is not a variable")
	}
}
