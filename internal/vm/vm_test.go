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

// runUnchecked compiles and runs src without failing on a check error, so a
// program the batch checker rejects still reaches the VM. The batch checker now
// rejects a missing return (the termination rule is shared with chip run via
// internal/typerules), so runProgram would stop at that error and never exercise
// the VM's own trap. check.Check populates Info even when it returns an error, so
// the compiler still has the type data it needs. This path stands in for an
// embedding host that compiles without checking, or checks more leniently.
func runUnchecked(src string) (string, error) {
	p, err := parser.New(strings.NewReader(src))
	if err != nil {
		return "", err
	}
	f, err := p.Parse()
	if err != nil {
		return "", err
	}
	info, _ := check.Check(f) // ignore check errors on purpose; Info is still populated
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

// A function with a declared result that falls off the end without returning
// must fault with a clean runtime error, never panic. Both checkers now reject a
// missing return up front, so this drives the VM through runUnchecked — standing
// in for an embedding host that compiles without checking. Before the
// OpMissingReturn trap the caller popped a return value that was never pushed,
// underflowing the stack and panicking the host with "index out of range [-1]".
// A panic here crashes the test binary, so this also guards against regressing
// the trap.
func TestMissingReturnTrapsCleanly(t *testing.T) {
	cases := map[string]string{
		"fall through": `package main
func main() { print(f()) }
func f() int { print(1) }`,
		"only one branch returns": `package main
func main() { print(g(0)) }
func g(x int) int {
    if x > 0 {
        return x
    }
}`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := runUnchecked(src)
			if err == nil || !strings.Contains(err.Error(), "missing return") {
				t.Fatalf("expected a missing-return runtime error, got %v", err)
			}
		})
	}
}

// The arith and compare paths carry default arms for operand/operator pairs the
// checker forbids: % on floats, a non-+ operator on strings, and ordering bools.
// chip's own checker rejects all three, so they are unreachable through chip run
// and chip.Compile; this drives them through runUnchecked, standing in for an
// embedding host that compiled without checking. Each must fault with a clean
// runtime error rather than silently push a wrong value — and, for float %,
// rather than push nothing, underflow the operand stack, and panic the host.
func TestVMArmsRejectCheckerImpossibleOps(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string // the operand-type the message must name
	}{
		{"float modulo", `package main
func main() { print(1.0 % 2.0) }`, "for float"},
		{"string subtraction", `package main
func main() { print("a" - "b") }`, "for string"},
		{"ordered bools", `package main
func main() { print((1 < 2) < (3 < 4)) }`, "for bool"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runUnchecked(tc.src)
			if err == nil {
				t.Fatalf("expected a runtime error, got none (out=%q)", out)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.want)
			}
			if out != "" {
				t.Errorf("output = %q, want empty (fault before print)", out)
			}
		})
	}
}

// A function that does return on every path is unaffected: the trailing trap is
// unreachable and the program runs normally.
func TestResultFunctionReturnsNormally(t *testing.T) {
	got := mustRun(t, `package main
func main() { print(pick(1)) }
func pick(x int) int {
    if x > 0 {
        return x
    }
    return 0
}`)
	if got != "1\n" {
		t.Errorf("got %q, want %q", got, "1\n")
	}
}
