package lint_test

import (
	"strings"
	"testing"

	"github.com/jackspirou/chip/internal/check"
	"github.com/jackspirou/chip/internal/diag"
	"github.com/jackspirou/chip/internal/lint"
	"github.com/jackspirou/chip/internal/parser"
)

func lintSrc(t *testing.T, src string) lint.IssueList {
	t.Helper()
	p, err := parser.New(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	f, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	info, err := check.Check(f)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	return lint.Lint(f, info)
}

// lintNoCheck parses and lints src directly with empty type info, bypassing the
// batch type checker. It is for programs the batch checker cannot yet handle —
// a used import resolves to a qualified call the checker rejects (Slice 5) — so
// the purely syntactic checks (e.g. unused imports) can still be exercised.
func lintNoCheck(t *testing.T, src string) lint.IssueList {
	t.Helper()
	p, err := parser.New(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	f, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return lint.Lint(f, &check.Info{})
}

func hasIssue(issues lint.IssueList, substr string) bool {
	for _, i := range issues {
		if strings.Contains(i.Msg, substr) {
			return true
		}
	}
	return false
}

func TestLintClean(t *testing.T) {
	// sum is defined before it is used, so there is no use-before-def hint.
	issues := lintSrc(t, `package main
func sum(xs []int) int {
    total := 0
    i := 0
    for i < len(xs) {
        total = total + xs[i]
        i = i + 1
    }
    return total
}
func main() { print(sum([]int{1, 2, 3})) }`)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
}

func TestLintUseBeforeDef(t *testing.T) {
	// a calls b, but b is defined after a.
	issues := lintSrc(t, `package main
func a() int { return b() }
func b() int { return 42 }
func main() { print(a()) }`)
	if !hasIssue(issues, "b used before its definition") {
		t.Fatalf("expected use-before-def hint, got %v", issues)
	}
}

func TestLintUnusedVar(t *testing.T) {
	issues := lintSrc(t, `package main
func main() {
    x := 1
    print(2)
}`)
	if !hasIssue(issues, "x declared and not used") {
		t.Fatalf("expected unused var, got %v", issues)
	}
}

func TestLintWriteOnlyVar(t *testing.T) {
	issues := lintSrc(t, `package main
func main() {
    x := 1
    x = 2
    print(3)
}`)
	if !hasIssue(issues, "x declared and not used") {
		t.Fatalf("expected write-only var flagged, got %v", issues)
	}
}

func TestLintUnusedFunc(t *testing.T) {
	issues := lintSrc(t, `package main
func main() { print(1) }
func helper() int { return 2 }`)
	if !hasIssue(issues, "function helper is never used") {
		t.Fatalf("expected unused func, got %v", issues)
	}
}

func TestLintUnreachable(t *testing.T) {
	issues := lintSrc(t, `package main
func main() { print(f()) }
func f() int {
    return 1
    return 2
}`)
	if !hasIssue(issues, "unreachable code") {
		t.Fatalf("expected unreachable code, got %v", issues)
	}
}

// TestLintUnreachableCarriesMachineRepair checks that the unreachable-code issue
// ships a machine-applicable removal: applying its edit deletes the dead tail and
// leaves a clean, still-terminating function. This is the one lint fix `chip fix`
// applies automatically (removing provably-dead code never changes behavior).
func TestLintUnreachableCarriesMachineRepair(t *testing.T) {
	src := `package main
func main() { print(f()) }
func f() int {
    return 1
    return 2
}`
	issues := lintSrc(t, src)

	var found *lint.Issue
	for i := range issues {
		if issues[i].Msg == "unreachable code" {
			found = &issues[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("no unreachable-code issue, got %v", issues)
	}
	if found.Suggestion == nil {
		t.Fatal("unreachable issue carries no suggestion")
	}
	if found.Suggestion.Applicability != diag.Machine {
		t.Errorf("applicability = %q, want machine", found.Suggestion.Applicability)
	}
	if found.Repair == nil || found.Repair.FixID != "unreachable" {
		t.Errorf("repair = %+v, want FixID unreachable", found.Repair)
	}

	out, err := diag.ApplyEdits([]byte(src), found.Suggestion.Edit)
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}
	want := `package main
func main() { print(f()) }
func f() int {
    return 1
}`
	if string(out) != want {
		t.Errorf("after fix:\n%q\nwant:\n%q", out, want)
	}
}

// PE8 — an imported package that is never referenced is reported. The unused
// import type-checks clean (the checker ignores imports and there is no
// qualified call), so the normal lint path reaches it.
func TestLintUnusedImport(t *testing.T) {
	issues := lintSrc(t, `package main
import "geometry"
func main() { print(1) }`)
	if !hasIssue(issues, "imported and not used") {
		t.Fatalf("expected unused-import hint, got %v", issues)
	}
}

// An import referenced as a qualifier (pkg.Fn) is not flagged. The qualified
// call defeats the batch checker (Slice 5), so this lints the tree directly.
func TestLintUsedImportNotFlagged(t *testing.T) {
	issues := lintNoCheck(t, `package main
import "geometry"
func main() { print(geometry.Area(3, 4)) }`)
	if hasIssue(issues, "imported and not used") {
		t.Fatalf("a used import should not be flagged, got %v", issues)
	}
}

// An aliased import is judged by its alias: using the alias clears it, and the
// original path's base name does not.
func TestLintUnusedImportAlias(t *testing.T) {
	used := lintNoCheck(t, `package main
import g "geometry"
func main() { print(g.Area(3, 4)) }`)
	if hasIssue(used, "imported and not used") {
		t.Fatalf("alias used as qualifier should not be flagged, got %v", used)
	}
	unused := lintNoCheck(t, `package main
import g "geometry"
func main() { print(geometry.Area(3, 4)) }`)
	if !hasIssue(unused, "imported and not used") {
		t.Fatalf("alias never used should be flagged, got %v", unused)
	}
}

// With the batch checker's import awareness (Slice 5), a used import lints clean
// through the full check+lint path: neither an unused-import hint nor a spurious
// "undefined" from the qualified call.
func TestLintUsedImportFullPath(t *testing.T) {
	issues := lintSrc(t, `package main
import "geometry"
func main() { print(geometry.Area(3, 4)) }`)
	if hasIssue(issues, "imported and not used") {
		t.Fatalf("a used import should not be flagged, got %v", issues)
	}
	if hasIssue(issues, "undefined") {
		t.Fatalf("the qualified call should not be reported undefined, got %v", issues)
	}
}
