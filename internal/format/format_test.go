package format_test

import (
	"strings"
	"testing"

	"github.com/jackspirou/chip/internal/format"
)

func fmtSrc(t *testing.T, src string) string {
	t.Helper()
	out, err := format.Source([]byte(src))
	if err != nil {
		t.Fatalf("format: %v", err)
	}
	return string(out)
}

func TestFormatCanonical(t *testing.T) {
	in := "package main\nfunc  add(x int,y int)int{\nreturn x+y\n}"
	want := "package main\n\nfunc add(x int, y int) int {\n    return x + y\n}\n"
	if got := fmtSrc(t, in); got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestFormatIdempotent(t *testing.T) {
	// Canonical form lists definitions in dependency order (a callee before its
	// caller), so sum precedes main.
	src := `package main

func sum(xs []int) int {
    total := 0
    i := 0
    for i < len(xs) {
        total = total + xs[i]
        i = i + 1
    }
    return total
}

func main() {
    xs := []int{3, 1, 4}
    print(sum(xs))
}
`
	once := fmtSrc(t, src)
	if once != src {
		t.Errorf("not idempotent on canonical form:\ngot:\n%s", once)
	}
	if twice := fmtSrc(t, once); twice != once {
		t.Errorf("format not stable:\nonce:\n%s\ntwice:\n%s", once, twice)
	}
}

func TestFormatPrecedenceParens(t *testing.T) {
	cases := map[string]string{
		"package main\nfunc main() { print((2 + 3) * 4) }": "print((2 + 3) * 4)",
		"package main\nfunc main() { print(2 + 3 * 4) }":   "print(2 + 3 * 4)",
		"package main\nfunc main() { print(a - (b - c)) }": "print(a - (b - c))",
		"package main\nfunc main() { print(a - b - c) }":   "print(a - b - c)",
		"package main\nfunc main() { print(-(a + b)) }":    "print(-(a + b))",
	}
	for in, wantLine := range cases {
		if got := fmtSrc(t, in); !strings.Contains(got, wantLine) {
			t.Errorf("for %q:\ngot:\n%s\nwant line: %q", in, got, wantLine)
		}
	}
}

func TestFormatPreservesComments(t *testing.T) {
	in := "// a greeting program\npackage main\n\nfunc main() {\n    // say hi\n    print(\"hi\")\n}\n"
	got := fmtSrc(t, in)
	if !strings.Contains(got, "// a greeting program") || !strings.Contains(got, "// say hi") {
		t.Errorf("comments not preserved:\n%s", got)
	}
}

// fmt-for-streamability: a callee used before its definition is hoisted above
// its caller.
func TestFormatHoistsDefinitions(t *testing.T) {
	in := "package main\nfunc a() int { return b() }\nfunc b() int { return 42 }\n"
	out := fmtSrc(t, in)
	ai, bi := strings.Index(out, "func a("), strings.Index(out, "func b(")
	if ai < 0 || bi < 0 {
		t.Fatalf("both functions should be present:\n%s", out)
	}
	if bi > ai {
		t.Errorf("b should be hoisted above its use in a:\n%s", out)
	}
	if twice := fmtSrc(t, out); twice != out {
		t.Errorf("not idempotent:\nonce:\n%s\ntwice:\n%s", out, twice)
	}
}

// A definition is hoisted above a top-level statement that uses it.
func TestFormatDefAboveStatementUse(t *testing.T) {
	out := fmtSrc(t, "print(f())\nfunc f() int { return 42 }\n")
	fi, ui := strings.Index(out, "func f("), strings.Index(out, "print(f())")
	if fi < 0 || ui < 0 {
		t.Fatalf("expected both the def and the use:\n%s", out)
	}
	if fi > ui {
		t.Errorf("the definition of f should be above its use:\n%s", out)
	}
	if twice := fmtSrc(t, out); twice != out {
		t.Errorf("not idempotent:\n%s", twice)
	}
}

// Top-level statements keep their original order.
func TestFormatNeverReordersStatements(t *testing.T) {
	out := fmtSrc(t, "print(1)\nprint(2)\nprint(3)\n")
	i1, i2, i3 := strings.Index(out, "print(1)"), strings.Index(out, "print(2)"), strings.Index(out, "print(3)")
	if i1 < 0 || i1 >= i2 || i2 >= i3 {
		t.Errorf("statements must keep their order:\n%s", out)
	}
}

// chip fmt canonicalizes the import block: sorted by path and deduplicated, and
// the result is stable under re-formatting (idempotent).
func TestFormatCanonicalizesImports(t *testing.T) {
	out := fmtSrc(t, "package main\nimport (\n    \"sort\"\n    \"fmt\"\n    \"fmt\"\n)\nfunc main() {}\n")
	fmtAt, sortAt := strings.Index(out, "\"fmt\""), strings.Index(out, "\"sort\"")
	if fmtAt < 0 || sortAt < 0 {
		t.Fatalf("both imports should be present:\n%s", out)
	}
	if fmtAt > sortAt {
		t.Errorf("imports should be sorted (fmt before sort):\n%s", out)
	}
	if n := strings.Count(out, "\"fmt\""); n != 1 {
		t.Errorf("the duplicate import should be removed, found %d copies of \"fmt\":\n%s", n, out)
	}
	if twice := fmtSrc(t, out); twice != out {
		t.Errorf("import formatting is not idempotent:\nonce:\n%s\ntwice:\n%s", out, twice)
	}
}

// A lone import (after dedup) collapses to a single line, never a group.
func TestFormatSingleImportOneLine(t *testing.T) {
	out := fmtSrc(t, "package main\nimport (\n    \"fmt\"\n    \"fmt\"\n)\nfunc main() {}\n")
	if !strings.Contains(out, "import \"fmt\"\n") {
		t.Errorf("a single (deduped) import should be a one-liner:\n%s", out)
	}
	if strings.Contains(out, "import (") {
		t.Errorf("a single import should not use a group:\n%s", out)
	}
}

// An import alias is preserved and orders after a bare import of the same path.
func TestFormatPreservesImportAlias(t *testing.T) {
	out := fmtSrc(t, "package main\nimport g \"geometry\"\nfunc main() { print(g.Area(3, 4)) }\n")
	if !strings.Contains(out, "import g \"geometry\"") {
		t.Errorf("alias should be preserved:\n%s", out)
	}
	if twice := fmtSrc(t, out); twice != out {
		t.Errorf("aliased import not idempotent:\nonce:\n%s\ntwice:\n%s", out, twice)
	}
}
