package parser

import "github.com/jackspirou/chip/token"

// nextProgram parses the file (program) body.
func (p *Parser) nextProgram() {
	p.enter()

	p.scope.Open()

	for p.tok.Type != token.EOF {
		if p.tok.Type == token.FUNC {
			p.nextFunctionDeclaration()
		} else {
			p.nextStatement()
		}
	}

	p.scope.Open()

	p.exit()
}
