package token

// A source position is represented by a Position value.
// A position is valid if Line > 0.
type Position struct {
	Line     int    // line number, starting at 1
	Column   int    // column number, starting at 1 (character count per line)
}

func NewPosition(line int, column int) Position {
	return Position{Line: line, Column: column,}
}

// IsValid returns true if the position is valid.
func (pos Position) IsValid() bool { return pos.Line > 0 }
