package parser

import (
	"github.com/jackspirou/chip/token"
)

// nextReturn parses a return.
func (p *Parser) nextReturn() {
	p.enterNext()

	p.nextExpected(token.RETURN)
	if p.tok.Type != token.RBRACE {
		p.nextExpression()
	}

	// exiting the parser's debug scope
	p.exitNext()
}
