package parser

import "github.com/jackspirou/chip/token"

// nextReturn parses a return.
func (p *Parser) nextReturn() {
	p.enter()
	p.nextExpected(token.RETURN)
	p.nextExpression()
	p.exit()
}
