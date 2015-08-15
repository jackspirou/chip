package parser

import "github.com/jackspirou/chip/token"

// nextProcedureBody parse a procedure body.
func (p *Parser) nextProcedureBody() {
	p.enter()
	p.nextExpected(token.LBRACE)
	p.nextStatement()
	for p.tok.Type != token.RBRACE {
		p.nextStatement()
	}
	p.nextExpected(token.RBRACE)
	p.exit()
}
