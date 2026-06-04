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
