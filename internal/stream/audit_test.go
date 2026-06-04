package stream

import (
	"bytes"
	"strings"
	"testing"
)

// mustError runs src and requires the error message to contain want.
func mustError(t *testing.T, src, want string) {
	t.Helper()
	out, err := runChip(t, src)
	if err == nil {
		t.Fatalf("expected an error containing %q, got nil (stdout %q)", want, out)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want it to contain %q", err, want)
	}
}

// Type-checker soundness fixes (from the correctness audit): values the checker
// once let through to the trusting executor are now rejected.
func TestCheckerSoundness(t *testing.T) {
	cases := []struct {
		name, src, want string
	}{
		{
			name: "missing return falls off the end",
			src:  "func greet() string { x := 1 }\nfunc main() { print(greet()) }",
			want: "missing return at end of function greet",
		},
		{
			name: "unknown type in signature",
			src:  "func f(n integer) int { return 99 }\nfunc main() { print(f(1)) }",
			want: "undefined type: integer",
		},
		{
			name: "slice equality",
			src:  "func main() { a := []int{1}\nb := []int{1}\nif a == b { print(1) } }",
			want: "slices are not comparable",
		},
		{
			name: "duplicate function",
			src:  "func f() int { return 1 }\nfunc f() int { return 2 }\nfunc main() { print(f()) }",
			want: "f redeclared",
		},
		{
			name: "duplicate := in a block",
			src:  "func main() { x := 1\nx := 2\nprint(x) }",
			want: "x redeclared in this block",
		},
		{
			name: "main with parameters",
			src:  "func main(n int) { print(n) }",
			want: "main must take no arguments",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			mustError(t, tt.src, tt.want)
		})
	}
}

// Runtime error paths are reached and reported (the executor's only guards).
func TestRuntimeErrors(t *testing.T) {
	cases := []struct {
		name, src, want string
	}{
		{"division by zero", "func main() { print(1 / 0) }", "division by zero"},
		{"modulo by zero", "func main() { print(1 % 0) }", "division by zero"},
		{"index out of range", "func main() { xs := []int{1, 2}\nprint(xs[5]) }", "index out of range"},
		{"runaway recursion", "func loop() int { return loop() }\nfunc main() { print(loop()) }", "call stack too deep"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			mustError(t, tt.src, tt.want)
		})
	}
}

// Slices have reference semantics: an indexed assignment mutates the backing
// array in place.
func TestSliceMutation(t *testing.T) {
	out, err := runChip(t, "func main() { xs := []int{1, 2, 3}\nxs[0] = 100\nprint(xs[0])\nprint(len(xs)) }")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "100\n3\n" {
		t.Fatalf("stdout = %q, want %q", out, "100\n3\n")
	}
}

// An empty, whitespace-only, or comment-only source runs cleanly with no output.
func TestEmptySource(t *testing.T) {
	for _, src := range []string{"", "   \n\t\n", "// just a comment\n"} {
		out, err := runChip(t, src)
		if err != nil {
			t.Fatalf("src %q: unexpected error: %v", src, err)
		}
		if out != "" {
			t.Fatalf("src %q: stdout = %q, want empty", src, out)
		}
	}
}

// The REPL may redefine a function across inputs (duplicate detection is
// per-source, not per-session).
func TestREPLRedefineAcrossInputs(t *testing.T) {
	in := strings.NewReader("func f() int { return 1 }\nprint(f())\nfunc f() int { return 2 }\nprint(f())\n")
	var out bytes.Buffer
	if err := REPL(in, &out); err != nil {
		t.Fatalf("REPL: %v", err)
	}
	if out.String() != "1\n2\n" {
		t.Fatalf("stdout = %q, want %q", out.String(), "1\n2\n")
	}
}

// A bad REPL line is reported but does not end the session; the next line still
// runs.
func TestREPLRecoversFromError(t *testing.T) {
	in := strings.NewReader("print(nope())\nprint(\"ok\")\n")
	var out bytes.Buffer
	if err := REPL(in, &out); err != nil {
		t.Fatalf("REPL: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "undefined: nope") {
		t.Fatalf("expected the error to be reported, got %q", got)
	}
	if !strings.Contains(got, "ok\n") {
		t.Fatalf("expected the next line to run, got %q", got)
	}
}

// A sweep of core language behaviors, to keep evaluation stable as the engine
// changes.
func TestLanguageFeatures(t *testing.T) {
	cases := []struct {
		name, src, want string
	}{
		{"float arithmetic", "func main() { print(3.0 / 2.0) }", "1.5\n"},
		{"string concatenation", "func main() { print(\"a\" + \"b\" + \"c\") }", "abc\n"},
		{"modulo", "func main() { print(17 % 5) }", "2\n"},
		{"unary minus", "func main() { print(-7) }", "-7\n"},
		{"unary not", "func main() { if !(1 == 2) { print(3) } }", "3\n"},
		{"logical and", "func main() { if 1 == 1 && 2 == 2 { print(1) } }", "1\n"},
		{"logical or", "func main() { if 1 == 2 || 3 == 3 { print(2) } }", "2\n"},
		{
			name: "if / else if / else",
			src:  "func sign(n int) int { if n < 0 { return 0 } else if n == 0 { return 1 } return 2 }\nfunc main() { print(sign(7)) }",
			want: "2\n",
		},
		{
			name: "for loop accumulation",
			src:  "func main() { sum := 0\ni := 1\nfor i <= 5 { sum = sum + i\ni = i + 1 }\nprint(sum) }",
			want: "15\n",
		},
		{
			name: "nested scope shadowing",
			src:  "func main() { x := 1\nif 1 == 1 { x := 2\nprint(x) }\nprint(x) }",
			want: "2\n1\n",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			out, err := runChip(t, tt.src)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != tt.want {
				t.Fatalf("stdout = %q, want %q", out, tt.want)
			}
		})
	}
}
