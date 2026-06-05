package diag

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"

	"github.com/jackspirou/chip/internal/scanner"
	"github.com/jackspirou/chip/internal/token"
)

// exampleFiles returns every .chp program under examples/ — the corpus the 3.0a
// offset mapping must round-trip for (tracker 3.0a).
func exampleFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	root := filepath.Join("..", "..", "examples")
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".chp" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk examples: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no example .chp files found")
	}
	return files
}

// runeBoundaries lists every byte offset that begins a rune, plus the end-of-source
// offset (len(src)), which is a valid one-past position.
func runeBoundaries(src []byte) []int {
	offs := []int{}
	for off := 0; off < len(src); {
		offs = append(offs, off)
		_, size := utf8.DecodeRune(src[off:])
		if size <= 0 {
			size = 1
		}
		off += size
	}
	return append(offs, len(src))
}

// TestOffsetPositionRoundTripExamples is the 3.0a acceptance check: for every
// rune boundary in every example, offset -> (line,col) -> offset is the identity.
func TestOffsetPositionRoundTripExamples(t *testing.T) {
	for _, path := range exampleFiles(t) {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		f := NewFile(src)
		offs := runeBoundaries(src)
		for _, off := range offs {
			line, col, ok := f.Position(off)
			if !ok {
				t.Fatalf("%s: Position(%d) not ok", path, off)
			}
			got, ok := f.Offset(line, col)
			if !ok {
				t.Fatalf("%s: Offset(%d,%d) not ok (from offset %d)", path, line, col, off)
			}
			if got != off {
				t.Fatalf("%s: round-trip offset %d -> (%d,%d) -> %d", path, off, line, col, got)
			}
		}
		t.Logf("%s: %d positions round-tripped", filepath.Base(path), len(offs))
	}
}

// TestOffsetMatchesScannerTokens proves Offset maps real scanner positions onto
// the exact source bytes the token was scanned from.
func TestOffsetMatchesScannerTokens(t *testing.T) {
	for _, path := range exampleFiles(t) {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		f := NewFile(src)
		s, err := scanner.New(bytes.NewReader(src))
		if err != nil {
			t.Fatalf("%s: scanner.New: %v", path, err)
		}
		checked := 0
		for {
			tok := s.Next()
			if tok.Type == token.EOF {
				break
			}
			if tok.Type == token.ERROR {
				t.Fatalf("%s: scan error: %s", path, tok.String())
			}
			off, ok := f.Offset(tok.Line(), tok.Column())
			if !ok {
				t.Fatalf("%s: Offset(%d,%d) not ok for %q", path, tok.Line(), tok.Column(), tok.String())
			}
			switch tok.Type {
			case token.IDENT, token.INT, token.FLOAT:
				// the literal is the verbatim source text; it must sit at off.
				if !bytes.HasPrefix(src[off:], []byte(tok.String())) {
					t.Fatalf("%s: token %q at (%d,%d) offset %d lands on %q",
						path, tok.String(), tok.Line(), tok.Column(), off, snippet(src, off))
				}
				checked++
			case token.STRING:
				// the token starts at the opening quote (the literal omits quotes).
				if off >= len(src) || src[off] != '"' {
					t.Fatalf("%s: STRING at (%d,%d) offset %d not at opening quote (%q)",
						path, tok.Line(), tok.Column(), off, snippet(src, off))
				}
				checked++
			}
		}
		t.Logf("%s: %d token offsets verified against source", filepath.Base(path), checked)
	}
}

// TestOffsetRuneColumns verifies that rune-based columns translate to byte
// offsets that account for multi-byte runes (the property that makes byte offsets
// distinct from columns in the first place).
func TestOffsetRuneColumns(t *testing.T) {
	// byte layout of "ab\nαβ\n": a=0 b=1 \n=2 α=3,4 β=5,6 \n=7 (len 8); α and β
	// are 2-byte runes.
	src := []byte("ab\nαβ\n")
	f := NewFile(src)
	cases := []struct{ line, col, wantOffset int }{
		{1, 1, 0}, // a
		{1, 2, 1}, // b
		{1, 3, 2}, // end of line 1 (the '\n')
		{2, 1, 3}, // α
		{2, 2, 5}, // β — rune column 2, but byte offset 5 because α is 2 bytes
		{2, 3, 7}, // end of line 2 (the '\n')
		{3, 1, 8}, // empty last line / end of source
	}
	for _, c := range cases {
		got, ok := f.Offset(c.line, c.col)
		if !ok || got != c.wantOffset {
			t.Fatalf("Offset(%d,%d) = %d,%v; want %d,true", c.line, c.col, got, ok, c.wantOffset)
		}
		gl, gc, ok := f.Position(c.wantOffset)
		if !ok || gl != c.line || gc != c.col {
			t.Fatalf("Position(%d) = (%d,%d),%v; want (%d,%d)", c.wantOffset, gl, gc, ok, c.line, c.col)
		}
	}
}

// TestOffsetCRLF pins the "\r\n" line model in NewFile: a CRLF pair is one line
// break, not two, so line 2 begins after the whole "\r\n". With the pre-fix
// double-count (every '\r' and '\n' starting a line) these offsets would be off by
// the number of preceding CRLFs. Mirrors TestOffsetRuneColumns for the terminator.
func TestOffsetCRLF(t *testing.T) {
	// byte layout of "ab\r\ncd\r\n": a=0 b=1 \r=2 \n=3 c=4 d=5 \r=6 \n=7 (len 8).
	src := []byte("ab\r\ncd\r\n")
	f := NewFile(src)
	if got := f.Lines(); got != 3 {
		t.Fatalf("Lines() = %d, want 3 (two CRLF breaks, then an empty last line)", got)
	}
	cases := []struct{ line, col, wantOffset int }{
		{1, 1, 0}, // a
		{1, 2, 1}, // b
		{1, 3, 2}, // end of line 1, addressing the '\r'
		{2, 1, 4}, // c — line 2 begins after the whole "\r\n", at offset 4 (not 3)
		{2, 2, 5}, // d
		{2, 3, 6}, // end of line 2, addressing the '\r'
		{3, 1, 8}, // empty last line / end of source
	}
	for _, c := range cases {
		got, ok := f.Offset(c.line, c.col)
		if !ok || got != c.wantOffset {
			t.Fatalf("Offset(%d,%d) = %d,%v; want %d,true", c.line, c.col, got, ok, c.wantOffset)
		}
		gl, gc, ok := f.Position(c.wantOffset)
		if !ok || gl != c.line || gc != c.col {
			t.Fatalf("Position(%d) = (%d,%d),%v; want (%d,%d)", c.wantOffset, gl, gc, ok, c.line, c.col)
		}
	}
}

// TestCRLFScannerOffsetAgree is the cross-check the soundness audit asked for: the
// scanner's line counting (skipSpaces) and NewFile must agree on the "\r\n" model,
// or a (line, column) carried by a diagnostic would convert to the wrong byte and
// chip fix would splice at the wrong place. It scans a CRLF program and, for every
// token, requires the offset NewFile derives from the token's (line, column) to
// equal the byte offset the scanner recorded directly (token.Offset, byte-exact
// since 3.0b). It also pins absolute line numbers, so the test fails if either
// side reverts to counting a CRLF as two lines.
func TestCRLFScannerOffsetAgree(t *testing.T) {
	// A small program with CRLF terminators. Lines: 1 package, 2 func, 3 x:=1,
	// 4 print(x), 5 closing brace.
	src := []byte("package main\r\nfunc main() {\r\n\tx := 1\r\n\tprint(x)\r\n}\r\n")
	f := NewFile(src)
	s, err := scanner.New(bytes.NewReader(src))
	if err != nil {
		t.Fatalf("scanner.New: %v", err)
	}
	// First-occurrence line for two identifiers that sit after 2 and 3 CRLFs; under
	// the old double-count they would report lines 5 and 7 instead of 3 and 4.
	wantLine := map[string]int{"x": 3, "print": 4}
	seen := map[string]bool{}
	tokens := 0
	for {
		tk := s.Next()
		if tk.Type == token.EOF {
			break
		}
		if tk.Type == token.ERROR {
			t.Fatalf("scan error: %s at %d:%d", tk.String(), tk.Line(), tk.Column())
		}
		tokens++
		// The two fixes must agree: the offset NewFile derives from this token's
		// (line, column) equals the offset the scanner recorded for the same token.
		off, ok := f.Offset(tk.Line(), tk.Column())
		if !ok {
			t.Fatalf("Offset(%d,%d) not ok for %q", tk.Line(), tk.Column(), tk.String())
		}
		if off != tk.Offset() {
			t.Fatalf("token %q (%d:%d): NewFile offset %d != scanner offset %d — skipSpaces and NewFile disagree on the CRLF line model",
				tk.String(), tk.Line(), tk.Column(), off, tk.Offset())
		}
		if tk.Type == token.IDENT && !seen[tk.String()] {
			if w, want := wantLine[tk.String()]; want {
				seen[tk.String()] = true
				if tk.Line() != w {
					t.Errorf("identifier %q on line %d, want %d (CRLF miscounted as two lines?)", tk.String(), tk.Line(), w)
				}
			}
		}
	}
	if tokens == 0 {
		t.Fatal("no tokens scanned")
	}
	for name, want := range wantLine {
		if !seen[name] {
			t.Errorf("identifier %q (expected on line %d) never scanned", name, want)
		}
	}
}

// TestOffsetOutOfRange checks that invalid positions and offsets report ok=false
// rather than returning a bogus value.
func TestOffsetOutOfRange(t *testing.T) {
	f := NewFile([]byte("ab\ncd")) // line 1 "ab", line 2 "cd"
	for _, b := range []struct{ line, col int }{
		{0, 1}, // line < 1
		{1, 0}, // col < 1
		{3, 1}, // line past end
		{1, 4}, // col past line 1 content (a,b, then end-of-line at col 3)
		{2, 4}, // col past line 2 content
	} {
		if off, ok := f.Offset(b.line, b.col); ok {
			t.Fatalf("Offset(%d,%d) = %d,true; want not ok", b.line, b.col, off)
		}
	}
	if _, _, ok := f.Position(-1); ok {
		t.Fatal("Position(-1) ok; want not ok")
	}
	if _, _, ok := f.Position(99); ok {
		t.Fatal("Position(99) ok; want not ok")
	}
}

// TestSpanApprox covers the 3.0a span approximation: endOffset = offset + byteLen,
// clamped to the source length.
func TestSpanApprox(t *testing.T) {
	src := []byte("foo := 12\n") // len 10
	f := NewFile(src)
	if off, end, ok := f.Span(1, 1, 3); !ok || off != 0 || end != 3 {
		t.Fatalf("Span(1,1,3) = %d,%d,%v; want 0,3,true", off, end, ok)
	}
	if _, end, ok := f.Span(1, 8, 100); !ok || end != len(src) {
		t.Fatalf("Span clamp end = %d,%v; want %d,true", end, ok, len(src))
	}
	if _, _, ok := f.Span(9, 1, 1); ok {
		t.Fatal("Span on missing line ok; want not ok")
	}
}

// snippet returns up to 12 bytes of src from off, for failure messages.
func snippet(src []byte, off int) string {
	if off < 0 || off >= len(src) {
		return ""
	}
	end := min(off+12, len(src))
	return string(src[off:end])
}
