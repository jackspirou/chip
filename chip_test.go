package chip_test

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jackspirou/chip"
)

func TestRun(t *testing.T) {
	src := `package main
func main() { print(gcd(252, 105)) }
func gcd(a int, b int) int {
    if b == 0 {
        return a
    }
    return gcd(b, a % b)
}`
	var buf bytes.Buffer
	if err := chip.Run([]byte(src), &buf); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := buf.String(); got != "21\n" {
		t.Errorf("got %q, want %q", got, "21\n")
	}
}

func TestCompileThenRun(t *testing.T) {
	prog, err := chip.Compile([]byte("package main\nfunc main() { print(1 + 2) }"))
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	var buf bytes.Buffer
	if err := prog.Run(&buf); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := buf.String(); got != "3\n" {
		t.Errorf("got %q, want %q", got, "3\n")
	}
}

func TestFormat(t *testing.T) {
	out, err := chip.Format([]byte("package main\nfunc main(){print( 1+2 )}"))
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if !strings.Contains(string(out), "print(1 + 2)") {
		t.Errorf("Format = %q", out)
	}
}

func TestCompileErrorDiagnostics(t *testing.T) {
	_, err := chip.Compile([]byte("package main\nfunc main() { print(x) }"))
	if err == nil {
		t.Fatal("expected an error")
	}
	var cerr *chip.Error
	if !errors.As(err, &cerr) {
		t.Fatalf("got %T, want *chip.Error", err)
	}
	if len(cerr.Diagnostics) == 0 || !strings.Contains(cerr.Diagnostics[0].Msg, "undefined: x") {
		t.Fatalf("diagnostics = %+v", cerr.Diagnostics)
	}
	if d := cerr.Diagnostics[0]; d.Line != 2 {
		t.Errorf("diagnostic line = %d, want 2", d.Line)
	}
}

func TestLint(t *testing.T) {
	ds, err := chip.Lint([]byte("package main\nfunc main() {\n    x := 1\n    print(2)\n}"))
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}
	found := false
	for _, d := range ds {
		if strings.Contains(d.Msg, "declared and not used") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected unused-variable diagnostic, got %+v", ds)
	}
}

func ExampleRun() {
	src := `package main
func main() { print(2 + 3) }`
	if err := chip.Run([]byte(src), os.Stdout); err != nil {
		panic(err)
	}
	// Output: 5
}
