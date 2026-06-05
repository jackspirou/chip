package check_test

import (
	"strings"
	"testing"

	"github.com/jackspirou/chip/internal/check"
	"github.com/jackspirou/chip/internal/parser"
)

// checkSrc parses and type-checks src, returning any type-checking error.
func checkSrc(t *testing.T, src string) error {
	t.Helper()
	p, err := parser.New(strings.NewReader(src))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	f, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, cerr := check.Check(f)
	return cerr
}

func mustCheck(t *testing.T, src string) {
	t.Helper()
	if err := checkSrc(t, src); err != nil {
		t.Fatalf("unexpected check error: %v", err)
	}
}

func wantErr(t *testing.T, src, substr string) {
	t.Helper()
	err := checkSrc(t, src)
	if err == nil {
		t.Fatalf("expected an error containing %q, got none", substr)
	}
	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("error = %q, want substring %q", err.Error(), substr)
	}
}

func TestCheckSimpleAddOK(t *testing.T) {
	mustCheck(t, `package main

func main() {
    three := 1 + 10
    return
}

func add(x int, y int) int {
    return x + y
}`)
}

func TestRecursionResolves(t *testing.T) {
	mustCheck(t, `package main

func gcd(a int, b int) int {
    if b == 0 {
        return a
    }
    return gcd(b, a % b)
}`)
}

func TestUndefinedVar(t *testing.T) {
	wantErr(t, "package main\nfunc main() { print(a) }", "undefined: a")
}

func TestReturnTypeMismatch(t *testing.T) {
	wantErr(t, `package main
func f() int { return "x" }`, "cannot return string as int")
}

func TestMissingReturnValue(t *testing.T) {
	wantErr(t, `package main
func f() int { return }`, "missing return value")
}

func TestNonBoolCondition(t *testing.T) {
	wantErr(t, `package main
func f() { if 1 { return } }`, "condition must be bool")
}

func TestCallArity(t *testing.T) {
	wantErr(t, `package main
func f(x int) {}
func g() { f(1, 2) }`, "wrong number of arguments")
}

func TestCallArgType(t *testing.T) {
	wantErr(t, `package main
func f(x int) {}
func g() { f("s") }`, "cannot use string as int")
}

func TestRedeclared(t *testing.T) {
	wantErr(t, `package main
func main() {
    x := 1
    x := 2
}`, "redeclared")
}

func TestBadOperandTypes(t *testing.T) {
	wantErr(t, `package main
func f() { x := 1 + "y" }`, "operator + is not defined")
}

// An imported package and a qualified call through it check clean: the batch
// checker treats the package opaquely (no Loader) rather than reporting the
// package name as undefined, so batch tooling works on programs with imports.
func TestCheckImportedPackageIsOpaque(t *testing.T) {
	mustCheck(t, `package main
import "geometry"
func main() { print(geometry.Area(3, 4)) }`)
}

// An aliased import binds the alias; a qualified call through it checks clean.
func TestCheckImportAliasIsOpaque(t *testing.T) {
	mustCheck(t, `package main
import g "geometry"
func main() { print(g.Area(3, 4)) }`)
}

// A truly undefined name (not an imported package) is still reported.
func TestCheckUndefinedStillErrors(t *testing.T) {
	wantErr(t, `package main
import "geometry"
func main() { print(nope.Area(3, 4)) }`, "undefined: nope")
}

// The tests below lock the four leniency gaps closed in the checker-soundness
// audit. Each program is one chip run (the streaming checker) rejects, so chip
// check (the batch checker) must reject it too: the invariant is that chip check
// is never more lenient than chip run. The operator, termination, and main-
// signature rules are shared with the streaming checker via internal/typerules,
// so these tests also guard the shared rules against drift.

// Bitwise and shift operators parse but are not part of the language chip runs.
// The batch checker once routed them through a default arm that returned int,
// silently accepting them; now it rejects every one, as the streaming checker
// and the executor do.
func TestBitwiseOperatorsRejected(t *testing.T) {
	for _, op := range []string{"&", "|", "^", "<<", ">>", "&^"} {
		src := "package main\nfunc main() {\n    x := 1 " + op + " 2\n    print(x)\n}"
		wantErr(t, src, "is not supported")
	}
}

// A bare function name used as a value (not called) is rejected. The batch
// checker once returned the function's signature here, which read as the invalid
// type and was quietly swallowed by callers; the streaming checker rejects it.
func TestFunctionAsValueRejected(t *testing.T) {
	wantErr(t, `package main
func f() int { return 1 }
func main() { print(f) }`, "f is not a value")
}

// A builtin is not a value either: print and len may only be called. (This
// behavior predates the audit; the test guards it across the checkCall refactor
// that routes builtin and function callees away from the value path.)
func TestBuiltinAsValueRejected(t *testing.T) {
	wantErr(t, `package main
func main() { print(len) }`, "len is not a value")
}

// A direct call to a function still checks clean — routing callees away from the
// value path must not break ordinary calls.
func TestFunctionCallStillOK(t *testing.T) {
	mustCheck(t, `package main
func add(a int, b int) int { return a + b }
func main() { print(add(2, 3)) }`)
}

// main must take no arguments. The streaming checker rejects a parameterized
// main; the batch checker did not, and now does.
func TestMainWithArgsRejected(t *testing.T) {
	wantErr(t, `package main
func main(x int) { print(x) }`, "main must take no arguments")
}

// A function with a declared result that can fall off the end is rejected at its
// closing brace. This was previously left to the linter and the streaming
// checker; the batch checker now enforces it, with the same message.
func TestMissingReturnRejected(t *testing.T) {
	wantErr(t, `package main
func f() int {
    print(1)
}
func main() { print(f()) }`, "missing return at end of function f")
}

// A function that returns on every path checks clean — the missing-return rule
// must not over-reject a well-formed function.
func TestTerminatingFunctionOK(t *testing.T) {
	mustCheck(t, `package main
func pick(x int) int {
    if x > 0 {
        return 1
    }
    return 0
}
func main() { print(pick(1)) }`)
}
