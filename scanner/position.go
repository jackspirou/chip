package scanner

// A source position is represented by a Position value.
// A position is valid if Line > 0.
type Position struct {
	Filename string // filename, if any
	Offset   int    // byte offset, starting at 0
	Line     int    // line number, starting at 1
	Column   int    // column number, starting at 1 (character count per line)
}

// IsValid returns true if the position is valid.
func (pos *Position) IsValid() bool { return pos.Line > 0 }

func (pos Position) String() string {
	s := pos.Filename
	if pos.IsValid() {
		if s != "" {
			s += ":"
		}
		s += fmt.Sprintf("%d:%d", pos.Line, pos.Column)
	}
	if s == "" {
		s = "???"
	}
	return s
}

// Pos returns the position of the character immediately after
// the character or token returned by the last call to Next or Scan.
/*
func (r *Reader) Pos() (pos Position) {
	pos.Filename = r.Filename
	pos.Offset = r.srcBufOffset + r.srcPos - r.lastCharLen
	switch {
	case r.column > 0:
		// common case: last character was not a '\n'
		pos.Line = r.line
		pos.Column = r.column
	case r.lastLineLen > 0:
		// last character was a '\n'
		pos.Line = r.line - 1
		pos.Column = r.lastLineLen
	default:
		// at the beginning of the source
		pos.Line = 1
		pos.Column = 1
	}
	return
}
*/
