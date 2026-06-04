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
	src := `package main

func main() {
    xs := []int{3, 1, 4}
    print(sum(xs))
}

func sum(xs []int) int {
    total := 0
    i := 0
    for i < len(xs) {
        total = total + xs[i]
        i = i + 1
    }
    return total
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
