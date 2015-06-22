package parser

import "github.com/jackspirou/chip/src/chip/token"

// Next Procedure Body. Parse a procedure body.
func (p *Parser) nextProcedureBody() {
	p.enter()
	p.nextExpected(token.LBRACE)
	p.nextStatement()
	for p.tok != token.RBRACE {
		p.nextStatement()
	}
	p.nextExpected(token.RBRACE)
	p.exit()
}
