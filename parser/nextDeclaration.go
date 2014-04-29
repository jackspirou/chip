package parser

import "github.com/jackspirou/chip/token"

// Next Declaration. Parse a declaration.
func (p *Parser) nextDeclaration() {
	p.enter()
	p.nextExpected(token.DEFINE)
	p.nextExpression()
	p.exit()
}
