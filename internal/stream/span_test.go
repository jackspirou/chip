package stream

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackspirou/chip/internal/parser"
	"github.com/jackspirou/chip/internal/scanner"
	"github.com/jackspirou/chip/internal/token"
)

// This is the slice 3.0b acceptance check: for a table of parse and type
// errors, the offending token named by the diagnostic's position has a
// byte-exact span. An agent reading "3:9" gets [Offset, End().Offset) and can
// replace exactly those bytes — endOffset-offset equals the offending token's
// length, including a string literal's surrounding quotes.

// diagPos returns the (line, column) of the first diagnostic in a streaming run
// error: a parser.ErrorList for parse errors or a stream.TypeError for type
// errors. Both are the position the offending token starts at.
func diagPos(t *testing.T, err error) token.Pos {
	t.Helper()
	if list, ok := errors.AsType[parser.ErrorList](err); ok {
		if len(list) == 0 {
			t.Fatal("empty parser.ErrorList")
		}
		return list[0].Pos
	}
	if te, ok := errors.AsType[TypeError](err); ok {
		return te.Pos
	}
	t.Fatalf("error %v (%T) carries no diagnostic position", err, err)
	return token.Pos{}
}

// tokenAt scans src and returns the token whose first character sits exactly at
// pos — the token a diagnostic at pos refers to.
func tokenAt(t *testing.T, src string, pos token.Pos) token.Token {
	t.Helper()
	s, err := scanner.New(strings.NewReader(src))
	if err != nil {
		t.Fatalf("scanner.New: %v", err)
	}
	for {
		tk := s.Next()
		if tk.Line() == pos.Line && tk.Column() == pos.Column {
			return tk
		}
		if tk.Type == token.EOF {
			t.Fatalf("no token at %d:%d in %q", pos.Line, pos.Column, src)
			return tk
		}
	}
}

// TestOffendingTokenSpans is the 3.0b Verify: every diagnostic position resolves
// to a token whose recorded span is exactly the offending source text.
func TestOffendingTokenSpans(t *testing.T) {
	cases := []struct {
		name, src, want string
	}{
		// Parse errors: the diagnostic points at the unexpected token.
		{"missing operand", "func main() { print(1 + ) }\n", ")"},
		{"missing close paren", "func main() { print(1 } }\n", "}"},
		// Type errors: the diagnostic points at the offending expression's first
		// token (an identifier, an operator, or a literal).
		{"undefined name", "func main() { print(foo) }\n", "foo"},
		{"binary operand mismatch", "func main() { print(1 + \"x\") }\n", "+"},
		{"return type mismatch", "func f() int { return \"x\" }\n", `"x"`},
		{"non-bool condition", "func main() { if 1 { print(2) } }\n", "1"},
		{"argument mismatch", "func f(n int) int { return n }\nfunc main() { print(f(\"hi\")) }\n", `"hi"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runChip(t, tc.src)
			if err == nil {
				t.Fatalf("expected an error for %q, got nil", tc.src)
			}
			pos := diagPos(t, err)
			tk := tokenAt(t, tc.src, pos)
			off, end := tk.Offset(), tk.End().Offset
			if off < 0 || end < off || end > len(tc.src) {
				t.Fatalf("span [%d,%d) out of range (len %d)", off, end, len(tc.src))
			}
			got := tc.src[off:end]
			if got != tc.want {
				t.Fatalf("offending token at %d:%d span = %q, want %q (err: %v)",
					pos.Line, pos.Column, got, tc.want, err)
			}
			if end-off != len(tc.want) {
				t.Fatalf("span length %d, want %d (token %q)", end-off, len(tc.want), got)
			}
		})
	}
}
