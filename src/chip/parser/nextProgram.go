package parser

import "github.com/jackspirou/chip/src/chip/token"

// Next Program. Parse the file (program) body.
func (p *Parser) nextProgram() {
	p.enter()
	for p.tok.Type != token.EOF {
		if p.tok.Type == token.FUNC {
			p.nextProcedure()
		} else {
			p.nextStatement()
		}
	}
	p.exit()
}
