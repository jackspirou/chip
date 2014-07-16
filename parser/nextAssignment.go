package parser

import "github.com/JackSpirou/chip/token"

// Next Assignment. Parse an assignment.
func (p *Parser) nextAssignment() {
	p.enter()
	p.nextExpected(token.ASSIGN)
	p.nextExpression()
	p.exit()
}
