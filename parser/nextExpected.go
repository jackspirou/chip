package parser

import (
	"github.com/jackspirou/chip/token"
)

// Next Expected. Expects argument as the next token.
func (p *Parser) nextExpected(expected token.Type) {
	if p.tok.Type == expected {
		p.next()
	} else {
		msg := "\"" + expected.String() + "\" expected instead of \"" + p.tok.String() + "\"."
		panic(msg)
	}
}
