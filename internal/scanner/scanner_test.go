package scanner_test

import (
	"strings"
	"testing"

	"github.com/jackspirou/chip/internal/scanner"
	"github.com/jackspirou/chip/internal/token"
)

type tok struct {
	typ  token.Type
	lit  string
	line int
	col  int
}

func scanAll(t *testing.T, src string) []tok {
	t.Helper()
	s, err := scanner.New(strings.NewReader(src))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var toks []tok
	for {
		tk := s.Next()
		if tk.Type == token.ERROR {
			t.Fatalf("scan error: %s at %d:%d", tk, tk.Line(), tk.Column())
		}
		toks = append(toks, tok{tk.Type, tk.String(), tk.Line(), tk.Column()})
		if tk.Type == token.EOF {
			return toks
		}
	}
}

func assertToks(t *testing.T, src string, want []tok) {
	t.Helper()
	got := scanAll(t, src)
	if len(got) != len(want) {
		t.Fatalf("%q: got %d tokens, want %d\n got: %v\nwant: %v", src, len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%q token %d = %+v, want %+v", src, i, got[i], want[i])
		}
	}
}

// TestScanCallPositions covers the historical off-by-one: single-character
// identifiers and operators/delimiters must point at their first character.
func TestScanCallPositions(t *testing.T) {
	assertToks(t, "print(a+b)", []tok{
		{token.IDENT, "print", 1, 1},
		{token.LPAREN, "(", 1, 6},
		{token.IDENT, "a", 1, 7},
		{token.ADD, "+", 1, 8},
		{token.IDENT, "b", 1, 9},
		{token.RPAREN, ")", 1, 10},
		{token.EOF, "EOF", 1, 11},
	})
}

func TestScanMultiline(t *testing.T) {
	assertToks(t, "x := 1\n  yy = 22", []tok{
		{token.IDENT, "x", 1, 1},
		{token.DEFINE, ":=", 1, 3},
		{token.INT, "1", 1, 6},
		{token.IDENT, "yy", 2, 3},
		{token.ASSIGN, "=", 2, 6},
		{token.INT, "22", 2, 8},
		{token.EOF, "EOF", 2, 10},
	})
}

func TestScanComparison(t *testing.T) {
	assertToks(t, "a == b", []tok{
		{token.IDENT, "a", 1, 1},
		{token.EQL, "==", 1, 3},
		{token.IDENT, "b", 1, 6},
		{token.EOF, "EOF", 1, 7},
	})
}

func TestScanIndex(t *testing.T) {
	assertToks(t, "xs[i]", []tok{
		{token.IDENT, "xs", 1, 1},
		{token.LBRACK, "[", 1, 3},
		{token.IDENT, "i", 1, 4},
		{token.RBRACK, "]", 1, 5},
		{token.EOF, "EOF", 1, 6},
	})
}

func TestScanStringAndComment(t *testing.T) {
	assertToks(t, `s := "hi"  // note`, []tok{
		{token.IDENT, "s", 1, 1},
		{token.DEFINE, ":=", 1, 3},
		{token.STRING, "hi", 1, 6},
		{token.COMMENT, "note", 1, 12},
		{token.EOF, "EOF", 1, 19},
	})
}
