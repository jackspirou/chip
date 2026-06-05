// Package diag turns chip's line/column source positions into byte offsets.
//
// chip records positions as token.Pos{Line, Column}: 1-based counts where Column
// is a *rune* index within its line — the scanner advances one column per rune
// (see internal/scanner). The agent-facing diagnostic schema needs byte offsets,
// because automated edits apply to bytes, not to (line, column) pairs.
//
// Offsets are computed from a position plus the source bytes, with no scanner
// change (plan slice 3.0a). A line begins at offset 0 and after each line break,
// where "\n", a lone "\r", and "\r\n" each count as one break — mirroring the
// scanner's end-of-line handling (see skipSpaces) so a token's Column maps back
// to the exact byte it was scanned from. File stays the way to convert a bare
// (line, column) — all a diagnostic message historically carries — into a byte
// offset.
//
// For byte-exact spans, prefer the offsets the scanner now records directly
// (token.Token.Offset/End) and the spans AST nodes expose (ast.Node.End), added
// in slice 3.0b: those are exact by construction. File.Span remains a length-based
// convenience for callers that hold only a (line, column) and a token length; the
// scanner/AST ends supersede it where a token or node is in hand.
//
// Caveat: a leading UTF-8 BOM, which the reader silently skips, is not accounted
// for in these computed offsets; chip sources do not use one, and the scanner's
// recorded offsets (3.0b) are unaffected since they count actual scanned bytes.
package diag

import (
	"sort"
	"unicode/utf8"
)

// File indexes a source buffer so positions and byte offsets convert in both
// directions. Construct one per source with NewFile; it precomputes line starts
// so Offset and Position cost O(log lines + runes-on-line).
type File struct {
	src    []byte
	starts []int // byte offset at which each line begins; starts[0] == 0
}

// NewFile indexes src for position/offset conversion.
func NewFile(src []byte) *File {
	starts := []int{0}
	for i := range len(src) {
		switch src[i] {
		case '\n':
			// End of line at '\n': a lone "\n" or the "\n" of a "\r\n" pair. In
			// the pair the next line begins after the '\n', so recording the start
			// here (and not at the '\r') keeps a "\r\n" break counting as one line,
			// mirroring the scanner (see skipSpaces).
			starts = append(starts, i+1)
		case '\r':
			// A lone '\r' (old-Mac line ending) ends a line; a "\r\n" pair is
			// handled by the '\n' case above, so skip the '\r' that precedes one.
			if i+1 >= len(src) || src[i+1] != '\n' {
				starts = append(starts, i+1)
			}
		}
	}
	return &File{src: src, starts: starts}
}

// Lines reports the number of indexed lines (always >= 1).
func (f *File) Lines() int { return len(f.starts) }

// Offset returns the 0-based byte offset of the 1-based (line, column) position.
// Column counts runes from the start of the line, exactly as token.Pos does, so
// a multi-byte rune earlier on the line shifts the byte offset accordingly.
//
// ok is false when the position is out of range: line < 1 or past the last line,
// column < 1, or column beyond the line's content. The position one past the
// final rune of a line is valid and addresses the line terminator (or, on the
// last line, end of source).
func (f *File) Offset(line, column int) (offset int, ok bool) {
	if line < 1 || line > len(f.starts) || column < 1 {
		return 0, false
	}
	off := f.starts[line-1]
	for c := 1; c < column; c++ {
		if off >= len(f.src) {
			return 0, false // column past end of source
		}
		r, size := utf8.DecodeRune(f.src[off:])
		if r == '\n' || r == '\r' {
			return 0, false // column past end of line
		}
		off += size
	}
	return off, true
}

// Position returns the 1-based (line, column) of a 0-based byte offset — the
// inverse of Offset. Column counts runes from the start of the line. ok is false
// if offset is negative, past the end of the source, or not on a rune boundary.
func (f *File) Position(offset int) (line, column int, ok bool) {
	if offset < 0 || offset > len(f.src) {
		return 0, 0, false
	}
	// line (1-based) = count of line starts at or before offset; starts[0] == 0
	// is always <= offset, so the search never returns 0.
	line = sort.Search(len(f.starts), func(i int) bool { return f.starts[i] > offset })
	column = 1
	for o := f.starts[line-1]; o < offset; {
		r, size := utf8.DecodeRune(f.src[o:])
		if r == utf8.RuneError && size <= 1 {
			return 0, 0, false // not a rune boundary / invalid UTF-8
		}
		o += size
		column++
	}
	return line, column, true
}

// Span returns the half-open byte range [offset, endOffset) for a token that
// starts at the 1-based (line, column) position and whose source text is byteLen
// bytes long: endOffset = offset + byteLen, clamped to the source length. It is a
// convenience for callers that hold only a (line, column) and a length; when a
// token or AST node is in hand, its Offset/End (slice 3.0b) give an exact span
// without the caller having to know the length. ok mirrors Offset.
func (f *File) Span(line, column, byteLen int) (offset, endOffset int, ok bool) {
	offset, ok = f.Offset(line, column)
	if !ok {
		return 0, 0, false
	}
	endOffset = min(offset+byteLen, len(f.src))
	return offset, endOffset, true
}
