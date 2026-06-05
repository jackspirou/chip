package chip_test

import (
	"strings"
	"testing"

	"github.com/jackspirou/chip"
)

// diagLines renders a report's diagnostics as "line:col: msg" strings.
func diagLines(r *chip.CheckReport) []string {
	out := make([]string, len(r.Diagnostics))
	for i, d := range r.Diagnostics {
		out[i] = d.String()
	}
	return out
}

func TestCheckClean(t *testing.T) {
	r := chip.Check([]byte("print(6 * 7)\n"))
	if !r.OK() {
		t.Fatalf("expected clean, got %v", diagLines(r))
	}
	if r.SuppressedCascades != 0 {
		t.Fatalf("clean source suppressed %d", r.SuppressedCascades)
	}
}

func TestCheckCleanWithFunc(t *testing.T) {
	r := chip.Check([]byte("func add(x int, y int) int { return x + y }\nprint(add(1, 2))\n"))
	if !r.OK() {
		t.Fatalf("expected clean, got %v", diagLines(r))
	}
}

// A3: collect every static error across the whole program — a return-type
// mismatch in a function body and an undefined name in a top-level statement —
// with no cascades to suppress.
func TestCheckCollectAllA3(t *testing.T) {
	r := chip.Check([]byte("func f(n int) string { return n }\nprint(undef)\n"))
	got := diagLines(r)
	want := []string{
		"1:31: cannot return int as string",
		"2:7: undefined: undef",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d diagnostics %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("diagnostic %d = %q, want %q", i, got[i], want[i])
		}
	}
	if r.SuppressedCascades != 0 {
		t.Fatalf("A3 suppressed %d, want 0", r.SuppressedCascades)
	}
}

// A crafted same-position parse cascade (a broken function signature) collapses
// to a single root diagnostic, and the phantom "undefined: x" the broken body
// provokes is suppressed too.
func TestCheckCascadeCollapses(t *testing.T) {
	r := chip.Check([]byte("func f( { print(x) }\n"))
	if len(r.Diagnostics) != 1 {
		t.Fatalf("got %d diagnostics %v, want 1 root", len(r.Diagnostics), diagLines(r))
	}
	if r.SuppressedCascades == 0 {
		t.Fatalf("expected suppressedCascades > 0, got 0")
	}
}

// Two independent broken constructs each keep their own root error; only the
// recovery artifacts between them are suppressed (genuine errors never dropped).
func TestCheckIndependentErrorsSurvive(t *testing.T) {
	r := chip.Check([]byte("print(1 +)\nprint(2 +)\n"))
	got := diagLines(r)
	if len(got) != 2 {
		t.Fatalf("got %d diagnostics %v, want 2 independent roots", len(got), got)
	}
	if !strings.HasPrefix(got[0], "1:") || !strings.HasPrefix(got[1], "2:") {
		t.Fatalf("expected one error per line, got %v", got)
	}
	if r.SuppressedCascades == 0 {
		t.Fatalf("expected the resync artifacts to be counted as suppressed")
	}
}

// A clean construct's type error survives even when another construct's syntax
// is broken (the broken one is pruned to its root; the independent type error is
// not collateral damage).
func TestCheckCleanConstructErrorSurvivesBrokenOne(t *testing.T) {
	r := chip.Check([]byte("print(1 +)\nfunc f(n int) string { return n }\n"))
	got := diagLines(r)
	want := []string{
		"1:10: expected expression, found )",
		"2:31: cannot return int as string",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d diagnostics %v, want %v", len(got), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("diagnostic %d = %q, want %q", i, got[i], want[i])
		}
	}
	if r.SuppressedCascades != 1 {
		t.Fatalf("suppressed %d, want 1 (the 2:1 resync artifact)", r.SuppressedCascades)
	}
}

// Top-level := may rebind a name, exactly as the streaming runtime allows, so a
// redefinition is not a "redeclared" error.
func TestCheckTopLevelRebindIsClean(t *testing.T) {
	r := chip.Check([]byte("x := 5\nx := 6\nprint(x)\n"))
	if !r.OK() {
		t.Fatalf("top-level rebind should check clean, got %v", diagLines(r))
	}
}
