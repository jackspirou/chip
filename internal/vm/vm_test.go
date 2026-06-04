package vm_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jackspirou/chip/internal/check"
	"github.com/jackspirou/chip/internal/compiler"
	"github.com/jackspirou/chip/internal/parser"
	"github.com/jackspirou/chip/internal/vm"
)

// runProgram compiles and runs src, returning its output and any error.
func runProgram(src string) (string, error) {
	p, err := parser.New(strings.NewReader(src))
	if err != nil {
		return "", err
	}
	f, err := p.Parse()
	if err != nil {
		return "", err
	}
	info, err := check.Check(f)
	if err != nil {
		return "", err
	}
	prog, err := compiler.Compile(f, info)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := vm.Run(prog, &buf); err != nil {
		return buf.String(), err
	}
	return buf.String(), nil
}

func mustRun(t *testing.T, src string) string {
	t.Helper()
	out, err := runProgram(src)
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	return out
}

func TestArithmetic(t *testing.T) {
	cases := map[string]string{
		"func main() { print(1 + 10) }":      "11\n",
		"func main() { print(2 * 3 + 4) }":   "10\n",
		"func main() { print((2 + 3) * 4) }": "20\n",
		"func main() { print(17 % 5) }":      "2\n",
		"func main() { print(-3 + 5) }":      "2\n",
		"func main() { print(10 - 2 - 3) }":  "5\n",
	}
	for body, want := range cases {
		src := "package main\n" + body
		if got := mustRun(t, src); got != want {
			t.Errorf("%s => %q, want %q", body, got, want)
		}
	}
}

func TestString(t *testing.T) {
	got := mustRun(t, "package main\nfunc main() { print(\"Hello, chip\") }")
	if got != "Hello, chip\n" {
		t.Errorf("got %q, want %q", got, "Hello, chip\n")
	}
}

func TestLocals(t *testing.T) {
	got := mustRun(t, `package main
func main() {
    three := 1 + 10
    print(three)
}`)
	if got != "11\n" {
		t.Errorf("got %q, want %q", got, "11\n")
	}
}

func TestAssign(t *testing.T) {
	got := mustRun(t, `package main
func main() {
    x := 1
    x = x + 41
    print(x)
}`)
	if got != "42\n" {
		t.Errorf("got %q, want %q", got, "42\n")
	}
}

func TestCall(t *testing.T) {
	got := mustRun(t, `package main
func main() { print(add(2, 3)) }
func add(x int, y int) int { return x + y }`)
	if got != "5\n" {
		t.Errorf("got %q, want %q", got, "5\n")
	}
}

func TestIfElse(t *testing.T) {
	got := mustRun(t, `package main
func main() {
    print(max(3, 7))
    print(max(9, 2))
}
func max(a int, b int) int {
    if a > b {
        return a
    }
    return b
}`)
	if got != "7\n9\n" {
		t.Errorf("got %q, want %q", got, "7\n9\n")
	}
}

func TestGCD(t *testing.T) {
	got := mustRun(t, `package main
func main() { print(gcd(252, 105)) }
func gcd(a int, b int) int {
    if b == 0 {
        return a
    }
    return gcd(b, a % b)
}`)
	if got != "21\n" {
		t.Errorf("got %q, want %q", got, "21\n")
	}
}

func TestDivideByZero(t *testing.T) {
	_, err := runProgram(`package main
func main() { print(1 / 0) }`)
	if err == nil || !strings.Contains(err.Error(), "division by zero") {
		t.Fatalf("expected division by zero error, got %v", err)
	}
}
