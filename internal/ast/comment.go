package ast

import "github.com/jackspirou/chip/internal/token"

// Comment is a single //-style or /* */-style comment. Text is the comment's
// content without its markers, so EndPos (the position one past the comment,
// recorded by the scanner) is kept explicitly rather than derived from Text.
type Comment struct {
	Slash  token.Pos
	EndPos token.Pos
	Text   string
}

func (c *Comment) Pos() token.Pos { return c.Slash }
func (c *Comment) End() token.Pos { return c.EndPos }
func (*Comment) node()            {}
