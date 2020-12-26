package parser

import "github.com/jackspirou/chip/token"

// nextAssignment parses the next assignment statement.
func (p *Parser) nextAssignment() {
	p.enterNext()

	p.nextExpected(token.ASSIGN)
	p.nextExpression()

	p.exitNext()
}
