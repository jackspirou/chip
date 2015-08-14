package parser

import "github.com/jackspirou/chip/src/chip/token"

// nextAssignment parses the next assignment statement.
func (p *Parser) nextAssignment() {
	p.enter()
	p.nextExpected(token.ASSIGN)
	p.nextExpression()
	p.exit()
}
