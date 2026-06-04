package stream

import (
	"bytes"
	"strings"
	"testing"
)

// runChip streams src through the engine, capturing stdout and the run error.
// A nil error corresponds to exit 0; a non-nil error to exit 1.
func runChip(t *testing.T, src string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := Run(strings.NewReader(src), &out)
	return out.String(), err
}

// E1 — a statement may call a function defined later in the stream; the call
// resolves by reading ahead.
func TestE1ForwardFuncRef(t *testing.T) {
	out, err := runChip(t, "print(f())\nfunc f() int { return 42 }\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "42\n" {
		t.Fatalf("stdout = %q, want %q", out, "42\n")
	}
}

// E2 — mutually recursive functions resolve because both register before main
// is called at EOF.
func TestE2MutualRecursion(t *testing.T) {
	const src = `func ping(n int) int { if n == 0 { return 100 } return pong(n - 1) }
func pong(n int) int { if n == 0 { return 200 } return ping(n - 1) }
func main() { print(ping(3)) }
`
	out, err := runChip(t, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "200\n" {
		t.Fatalf("stdout = %q, want %q", out, "200\n")
	}
}

// E3 — a forward variable reference cannot be satisfied (vars are
// define-before-use), so it errors with no output.
func TestE3ForwardVarUndefined(t *testing.T) {
	out, err := runChip(t, "print(x)\nx := 5\n")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "undefined: x") {
		t.Fatalf("error = %v, want it to contain %q", err, "undefined: x")
	}
	if out != "" {
		t.Fatalf("stdout = %q, want empty", out)
	}
}

// E5 — effects commit as they happen: the first print's output stands even
// though a later statement fails.
func TestE5PartialOutputThenError(t *testing.T) {
	out, err := runChip(t, "print(\"hello\")\nprint(nope())\n")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "undefined: nope") {
		t.Fatalf("error = %v, want it to contain %q", err, "undefined: nope")
	}
	if out != "hello\n" {
		t.Fatalf("stdout = %q, want %q", out, "hello\n")
	}
}

// E6 — a declaration-only program runs main at EOF, and main forward-references
// gcd.
func TestE6MainAtEOF(t *testing.T) {
	const src = `func main() { print(gcd(252, 105)) }
func gcd(a int, b int) int {
	if b == 0 {
		return a
	}
	return gcd(b, a % b)
}
`
	out, err := runChip(t, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "21\n" {
		t.Fatalf("stdout = %q, want %q", out, "21\n")
	}
}
