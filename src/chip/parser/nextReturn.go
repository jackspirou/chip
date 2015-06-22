package parser

import "github.com/jackspirou/chip/src/chip/token"

// Next Return. Parse a return.
func (p *Parser) nextReturn() {
	p.enter()
	p.nextExpected(token.RETURN)
	p.nextExpression()
	p.exit()
}
