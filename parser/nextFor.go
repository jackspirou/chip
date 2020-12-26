package parser

import (
	"github.com/jackspirou/chip/token"
)

// nextFor parses a for loop.
func (p *Parser) nextFor() error {
	p.enterNext()

	p.next() // Skip 'for' token
	p.nextExpression()
	p.nextExpected(token.LBRACE)
	p.nextStatement()
	p.nextExpected(token.RBRACE)

	p.exitNext()
	return nil
}
