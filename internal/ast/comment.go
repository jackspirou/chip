package ast

import "github.com/jackspirou/chip/internal/token"

// Comment is a single //-style or /* */-style comment. Text is the comment's
// content without its markers.
type Comment struct {
	Slash token.Pos
	Text  string
}

func (c *Comment) Pos() token.Pos { return c.Slash }
func (*Comment) node()            {}
