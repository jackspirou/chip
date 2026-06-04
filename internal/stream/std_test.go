package stream

import "testing"

// The standard library (written in chip, loaded before user code) is available
// to every program.
func TestStdlib(t *testing.T) {
	cases := []struct {
		name, src, want string
	}{
		{"max", "func main() { print(max(3, 7)) }", "7\n"},
		{"min", "func main() { print(min(3, 7)) }", "3\n"},
		{"abs of negative", "func main() { print(abs(-5)) }", "5\n"},
		{"abs of positive", "func main() { print(abs(5)) }", "5\n"},
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

// A user program may define its own function with the same name as a library
// function; the user definition wins.
func TestUserOverridesStdlib(t *testing.T) {
	out, err := runChip(t, "func max(a int, b int) int { return 0 }\nfunc main() { print(max(3, 7)) }\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "0\n" {
		t.Fatalf("stdout = %q, want %q", out, "0\n")
	}
}
