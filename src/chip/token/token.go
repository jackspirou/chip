package token

// Item represents a token returned from the scanner.
type Tok struct {
	typ Tokint // Type, such as itemNumber.
	lit string // Value, such as "23.2".
	pos Position
}

func NewTok(typ Tokint, lit string, pos Position) *Tok {
	return &Tok{typ: typ, lit: lit, pos: pos}
}

func NewEndTok() *Tok {
	typ := EOF
	lit := "EOF"
	pos := NewPosition(0, 0)
	return NewTok(typ, lit, pos)
}

func (t *Tok) String() string {
	return t.lit
}

func (t *Tok) Typ() Tokint {
	return t.typ
}

// A source position is represented by a Position value.
// A position is valid if Line > 0.
type Pos struct {
	Line   int // line number, starting at 1
	Column int // column number, starting at 1 (character count per line)
}

func NewPosition(line int, column int) Pos {
	return Pos{Line: line, Column: column}
}

// IsValid returns true if the position is valid.
func (pos Pos) IsValid() bool { return pos.Line > 0 }

// Lookup maps an identifier to its keyword token or IDENT (if not a keyword).
func Lookup(ident string) Tokint {
	if tok, is_keyword := keywords[ident]; is_keyword {
		return tok
	}
	return IDENT
}
