package scanner_test

import (
	"strings"
	"testing"

	"github.com/jackspirou/chip/internal/scanner"
	"github.com/jackspirou/chip/internal/token"
)

// goldenTok is one expected token in a golden stream: the type mnemonic
// (token.Type.String()), the literal (token.String()), the 1-based line and
// column, and the half-open byte span [off, end) the scanner recorded (token
// Offset and End().Offset, byte-exact since slice 3.0b). It carries byte spans
// and stores the type as a string, which is why it is distinct from the simpler
// tok used by the position tests in scanner_test.go.
type goldenTok struct {
	typ       string
	lit       string
	line, col int
	off, end  int
}

// scanGolden drives the scanner over src and returns the observed token stream in
// the same shape as the golden table, stopping after the first terminal token
// (EOF or ERROR). A safety cap turns any reintroduced scan-loop hang into a clear
// failure instead of a wedged test.
func scanGolden(t *testing.T, src string) []goldenTok {
	t.Helper()
	s, err := scanner.New(strings.NewReader(src))
	if err != nil {
		t.Fatalf("scanner.New(%q): %v", src, err)
	}
	var got []goldenTok
	const cap = 64
	for range cap {
		tk := s.Next()
		got = append(got, goldenTok{
			typ:  tk.Type.String(),
			lit:  tk.String(),
			line: tk.Line(), col: tk.Column(),
			off: tk.Offset(), end: tk.End().Offset,
		})
		if tk.Type == token.EOF || tk.Type == token.ERROR {
			return got
		}
	}
	t.Fatalf("scan did not terminate within %d tokens for %q", cap, src)
	return nil
}

// TestScanGolden pins the exact token stream — type, literal, line, column, and
// byte span — the scanner produces for a corpus of lexical edge cases. It
// replaces an earlier "probe" test that only logged its output and asserted
// nothing: a regression in number/string/comment/operator handling or in
// position/span math went undetected there but fails loudly here.
//
// Cases tagged KNOWN BUG pin behavior that is wrong but deliberate today; the
// comment says why. Fixing such a bug should update that one row on purpose,
// which is exactly the signal a golden test is meant to surface.
func TestScanGolden(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []goldenTok
	}{
		// --- numbers ---
		{
			// KNOWN BUG: "1.2.3" lexes as a single FLOAT — the number scanner does
			// not reject a second '.'. The parser later rejects the malformed value,
			// but the token itself spans all five bytes and claims to be a FLOAT.
			name: "multi-dot",
			src:  "1.2.3",
			want: []goldenTok{
				{"FLOAT", "1.2.3", 1, 1, 0, 5},
				{"EOF", "EOF", 1, 6, 5, 5},
			},
		},
		{
			// Trailing and leading dots are accepted (Go-style: 42. and .5).
			name: "trailing-dot",
			src:  "42.",
			want: []goldenTok{
				{"FLOAT", "42.", 1, 1, 0, 3},
				{"EOF", "EOF", 1, 4, 3, 3},
			},
		},
		{
			name: "leading-dot",
			src:  ".5",
			want: []goldenTok{
				{"FLOAT", ".5", 1, 1, 0, 2},
				{"EOF", "EOF", 1, 3, 2, 2},
			},
		},
		{
			// "." then a second "." that does not complete "...": the scanner is
			// mid-ellipsis and errors on the following digit.
			name: "double-dot-int",
			src:  "..5",
			want: []goldenTok{
				{"ERROR", "unexpected 5", 1, 3, 2, 2},
			},
		},
		{
			// A period between identifiers is a selector dot, not part of a number.
			name: "lone-period",
			src:  "a.b",
			want: []goldenTok{
				{"IDENT", "a", 1, 1, 0, 1},
				{".", ".", 1, 2, 1, 2},
				{"IDENT", "b", 1, 3, 2, 3},
				{"EOF", "EOF", 1, 4, 3, 3},
			},
		},
		{
			// "a" then ".." that starts an ellipsis but hits "b" instead.
			name: "two-dots",
			src:  "a..b",
			want: []goldenTok{
				{"IDENT", "a", 1, 1, 0, 1},
				{"ERROR", "unexpected b", 1, 4, 3, 3},
			},
		},
		{
			// Leading zeros are kept verbatim; there is no octal interpretation.
			name: "leading-zeros",
			src:  "007",
			want: []goldenTok{
				{"INT", "007", 1, 1, 0, 3},
				{"EOF", "EOF", 1, 4, 3, 3},
			},
		},
		{
			// The scanner does not range-check integers; overflow is a later concern.
			name: "huge-int",
			src:  "999999999999999999999999999999",
			want: []goldenTok{
				{"INT", "999999999999999999999999999999", 1, 1, 0, 30},
				{"EOF", "EOF", 1, 31, 30, 30},
			},
		},
		{
			// chip has no hex literals: "0x1F" splits into INT 0 and IDENT x1F.
			name: "hex-like",
			src:  "0x1F",
			want: []goldenTok{
				{"INT", "0", 1, 1, 0, 1},
				{"IDENT", "x1F", 1, 2, 1, 4},
				{"EOF", "EOF", 1, 5, 4, 4},
			},
		},
		{
			// No digit separators: "1_000" splits into INT 1 and IDENT _000.
			name: "underscore-num",
			src:  "1_000",
			want: []goldenTok{
				{"INT", "1", 1, 1, 0, 1},
				{"IDENT", "_000", 1, 2, 1, 5},
				{"EOF", "EOF", 1, 6, 5, 5},
			},
		},

		// --- strings ---
		{
			// Escape sequences are kept verbatim, not interpreted: the literal still
			// contains the backslash and 'n', and the closing quote is found normally.
			name: "str-escape-n",
			src:  "\"a\\nb\"",
			want: []goldenTok{
				{"STRING", "a\\nb", 1, 1, 0, 6},
				{"EOF", "EOF", 1, 7, 6, 6},
			},
		},
		{
			// KNOWN BUG: \" is not treated as an escaped quote. The string ends at the
			// backslash-quote (literal "a\"), leaving IDENT "b" and a dangling quote
			// that errors. Backslash escapes inside strings are not processed.
			name: "str-escape-quote",
			src:  "\"a\\\"b\"",
			want: []goldenTok{
				{"STRING", "a\\", 1, 1, 0, 4},
				{"IDENT", "b", 1, 5, 4, 5},
				{"ERROR", "string has no closing quote", 1, 6, 5, 6},
			},
		},
		{
			// A doubled backslash is kept verbatim and does not eat the closing quote.
			name: "str-backslash",
			src:  "\"a\\\\b\"",
			want: []goldenTok{
				{"STRING", "a\\\\b", 1, 1, 0, 6},
				{"EOF", "EOF", 1, 7, 6, 6},
			},
		},
		{
			// An unterminated string is a clean ERROR — not the infinite loop the
			// soundness audit fixed (see chip-lex-parse-outside-cap-envelope).
			name: "str-unterminated",
			src:  "\"abc",
			want: []goldenTok{
				{"ERROR", "string has no closing quote", 1, 1, 0, 4},
			},
		},
		{
			// Strings are single-line: a newline before the closing quote errors, with
			// the span ending at the newline.
			name: "str-newline",
			src:  "\"ab\ncd\"",
			want: []goldenTok{
				{"ERROR", "string has no closing quote", 1, 1, 0, 3},
			},
		},
		{
			name: "str-empty",
			src:  "\"\"",
			want: []goldenTok{
				{"STRING", "", 1, 1, 0, 2},
				{"EOF", "EOF", 1, 3, 2, 2},
			},
		},

		// --- comments ---
		//
		// KNOWN BUG (all block-comment cases below): nextComment leaves the closing
		// '/' of "*/" unconsumed, so every block comment is followed by a stray QUO
		// "/" token, and the COMMENT span ends one byte before the real "*/". This is
		// deliberately preserved to keep `chip run` byte-identical (plan A7) and is
		// deferred to its own slice; see the NOTE in scanner.go nextComment. The
		// practical effect is that an inline block comment is a parse error.
		{
			name: "block-inline",
			src:  "/* x */ 1",
			want: []goldenTok{
				{"COMMENT", "x ", 1, 1, 0, 6}, // span "/* x *" — misses the final '/'
				{"/", "/", 1, 7, 6, 7},        // stray QUO from the unconsumed '/'
				{"INT", "1", 1, 9, 8, 9},
				{"EOF", "EOF", 1, 10, 9, 9},
			},
		},
		{
			// An unterminated block comment is a clean ERROR (audit-fixed, like the
			// unterminated string).
			name: "block-unterminated",
			src:  "/* x",
			want: []goldenTok{
				{"ERROR", "comment never terminated", 1, 5, 4, 4},
			},
		},
		{
			// The COMMENT literal carries the embedded newline; the stray QUO follows.
			name: "block-multiline",
			src:  "/* a\nb */\n5",
			want: []goldenTok{
				{"COMMENT", "a\nb ", 1, 1, 0, 8},
				{"/", "/", 1, 9, 8, 9}, // stray QUO
				{"INT", "5", 2, 1, 10, 11},
				{"EOF", "EOF", 2, 2, 11, 11},
			},
		},
		{
			name: "block-at-eof",
			src:  "1 /* end */",
			want: []goldenTok{
				{"INT", "1", 1, 1, 0, 1},
				{"COMMENT", "end ", 1, 3, 2, 10},
				{"/", "/", 1, 11, 10, 11}, // stray QUO at end of input
				{"EOF", "EOF", 1, 12, 11, 11},
			},
		},
		{
			// Line comments are correct: the COMMENT spans to end-of-line and no stray
			// token follows.
			name: "line-comment-eof",
			src:  "1 // tail",
			want: []goldenTok{
				{"INT", "1", 1, 1, 0, 1},
				{"COMMENT", "tail", 1, 3, 2, 9},
				{"EOF", "EOF", 1, 10, 9, 9},
			},
		},
		{
			name: "block-empty",
			src:  "/**/",
			want: []goldenTok{
				{"COMMENT", "", 1, 1, 0, 3},
				{"/", "/", 1, 4, 3, 4}, // stray QUO
				{"EOF", "EOF", 1, 5, 4, 4},
			},
		},
		{
			name: "block-star-slash-x",
			src:  "/**/x",
			want: []goldenTok{
				{"COMMENT", "", 1, 1, 0, 3},
				{"/", "/", 1, 4, 3, 4}, // stray QUO
				{"IDENT", "x", 1, 5, 4, 5},
				{"EOF", "EOF", 1, 6, 5, 5},
			},
		},
		{
			name: "block-then-newline-comment",
			src:  "/* a */ // b",
			want: []goldenTok{
				{"COMMENT", "a ", 1, 1, 0, 6},
				{"/", "/", 1, 7, 6, 7}, // stray QUO between the two comments
				{"COMMENT", "b", 1, 9, 8, 12},
				{"EOF", "EOF", 1, 13, 12, 12},
			},
		},

		// --- operators and punctuation ---
		{
			name: "ellipsis",
			src:  "...",
			want: []goldenTok{
				{"...", "...", 1, 1, 0, 3},
				{"EOF", "EOF", 1, 4, 3, 3},
			},
		},
		{
			// KNOWN BUG (cosmetic): two dots at EOF report the EOF sentinel rune as the
			// Unicode replacement character in the message ("unexpected �").
			name: "two-dot-eof",
			src:  "..",
			want: []goldenTok{
				{"ERROR", "unexpected �", 1, 3, 2, 2},
			},
		},
		{
			name: "arrow",
			src:  "a <- b",
			want: []goldenTok{
				{"IDENT", "a", 1, 1, 0, 1},
				{"<-", "<-", 1, 3, 2, 4},
				{"IDENT", "b", 1, 6, 5, 6},
				{"EOF", "EOF", 1, 7, 6, 6},
			},
		},
		{
			name: "andnot",
			src:  "a &^ b",
			want: []goldenTok{
				{"IDENT", "a", 1, 1, 0, 1},
				{"&^", "&^", 1, 3, 2, 4},
				{"IDENT", "b", 1, 6, 5, 6},
				{"EOF", "EOF", 1, 7, 6, 6},
			},
		},
		{
			name: "andnotassign",
			src:  "a &^= b",
			want: []goldenTok{
				{"IDENT", "a", 1, 1, 0, 1},
				{"&^=", "&^=", 1, 3, 2, 5},
				{"IDENT", "b", 1, 7, 6, 7},
				{"EOF", "EOF", 1, 8, 7, 7},
			},
		},
		{
			name: "shlassign",
			src:  "a <<= b",
			want: []goldenTok{
				{"IDENT", "a", 1, 1, 0, 1},
				{"<<=", "<<=", 1, 3, 2, 5},
				{"IDENT", "b", 1, 7, 6, 7},
				{"EOF", "EOF", 1, 8, 7, 7},
			},
		},
		{
			name: "illegal-at",
			src:  "@",
			want: []goldenTok{
				{"ERROR", "unexpected '@'", 1, 1, 0, 0},
			},
		},

		// --- whitespace, line endings, unicode ---
		{
			name: "empty",
			src:  "",
			want: []goldenTok{
				{"EOF", "EOF", 1, 1, 0, 0},
			},
		},
		{
			// CRLF is one line break: "b" is on line 2 at byte offset 3 (after
			// "a\r\n"), not line 3. Guards the skipSpaces CRLF fix.
			name: "crlf",
			src:  "a\r\nb",
			want: []goldenTok{
				{"IDENT", "a", 1, 1, 0, 1},
				{"IDENT", "b", 2, 1, 3, 4},
				{"EOF", "EOF", 2, 2, 4, 4},
			},
		},
		{
			// A leading tab advances the column by one.
			name: "tab-indent",
			src:  "\ta",
			want: []goldenTok{
				{"IDENT", "a", 1, 2, 1, 2},
				{"EOF", "EOF", 1, 3, 2, 2},
			},
		},
		{
			// é is two bytes, so byte offsets and rune columns diverge: the IDENT spans
			// bytes [0,5) for four runes (columns 1..4), and ":=" follows at column 6.
			name: "unicode-ident",
			src:  "café := 1",
			want: []goldenTok{
				{"IDENT", "café", 1, 1, 0, 5},
				{":=", ":=", 1, 6, 6, 8},
				{"INT", "1", 1, 9, 9, 10},
				{"EOF", "EOF", 1, 10, 10, 10},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := scanGolden(t, c.src)
			if len(got) != len(c.want) {
				t.Fatalf("got %d tokens, want %d\n got: %+v\nwant: %+v", len(got), len(c.want), got, c.want)
			}
			for i, w := range c.want {
				if got[i] != w {
					t.Errorf("token %d mismatch:\n got %+v\nwant %+v", i, got[i], w)
				}
			}
		})
	}
}
