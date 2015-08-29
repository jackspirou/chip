package parser

import "github.com/jackspirou/chip/parser/token"

// nextReturn parses a return.
func (p *Parser) nextReturn() {
	p.enter()
	p.nextExpected(token.RETURN)
	p.nextExpression()
	p.exit()
}
