package parser

import (
	"github.com/JackSpirou/chip/token"
)

// Next Expected. Expects argument as the next token.
func (p *Parser) nextExpected(expected token.Tokint) {
	if p.tok == expected {
		p.next()
	} else {
		msg := "\"" + expected.String() + "\" expected instead of \"" + p.tokn.String() + "\"."
		panic(msg)
	}
}
