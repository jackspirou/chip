package stream

import (
	"strings"
	"testing"
)

// E4 — a return whose type disagrees with the declared result is rejected, even
// though nothing ever calls f (bodies are checked at registration).
func TestE4ReturnTypeMismatch(t *testing.T) {
	_, err := runChip(t, "func f() int { return \"x\" }\n")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot return string as int") {
		t.Fatalf("error = %v, want it to contain %q", err, "cannot return string as int")
	}
}

// Type-error programs must be rejected with the expected message.
func TestTypeErrorsReported(t *testing.T) {
	cases := []struct {
		name, src, want string
	}{
		{
			name: "return mismatch",
			src:  `func f() int { return "x" }`,
			want: "cannot return string as int",
		},
		{
			name: "argument mismatch",
			src:  "func f(n int) int { return n }\nfunc main() { print(f(\"x\")) }",
			want: "argument 1: cannot use string as int",
		},
		{
			name: "non-bool condition",
			src:  "func main() { if 1 { print(2) } }",
			want: "condition must be bool, got int",
		},
		{
			name: "binary operand mismatch",
			src:  "func main() { print(1 + \"x\") }",
			want: "operator + is not defined for int and string",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			out, err := runChip(t, tt.src)
			if err == nil {
				t.Fatalf("expected an error, got nil (stdout %q)", out)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

// The type-correct twin of each program above runs to completion (no false
// positives).
func TestTypeCorrectTwinsRun(t *testing.T) {
	cases := []struct {
		name, src, want string
	}{
		{
			name: "return ok",
			src:  "func f() int { return 7 }\nfunc main() { print(f()) }",
			want: "7\n",
		},
		{
			name: "argument ok",
			src:  "func f(n int) int { return n }\nfunc main() { print(f(5)) }",
			want: "5\n",
		},
		{
			name: "bool condition ok",
			src:  "func main() { if 1 == 1 { print(2) } }",
			want: "2\n",
		},
		{
			name: "binary operands ok",
			src:  "func main() { print(\"a\" + \"b\") }",
			want: "ab\n",
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
