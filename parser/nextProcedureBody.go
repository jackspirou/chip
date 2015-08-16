package parser

import "github.com/jackspirou/chip/token"

// nextFunctionBody parses a function body.
func (p *Parser) nextFunctionBody() {
	p.enter()
	p.nextExpected(token.LBRACE)
	p.nextStatement()
	for p.tok.Type != token.RBRACE {
		p.nextStatement()
	}
	p.nextExpected(token.RBRACE)
	p.exit()
}
