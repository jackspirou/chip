package parser

import (
	"github.com/jackspirou/chip/ast"
	"github.com/jackspirou/chip/token"
)

// nextProgram parses the file (program) body.
func (p *Parser) nextProgram() (program ast.Program, err error) {
	p.enterNext()

	p.scope.Open()
	for p.tok.Type != token.EOF {
		if p.tok.Type == token.FUNC {
			p.nextFunctionDeclaration()
			continue
		}
		if s, err := p.nextDeclaration(); err == nil {
			program.Statements = append(program.Statements, s)
			continue
		}
		if err != nil {
			panic(err)
		}
	}
	p.scope.Open() // why do we do this?  are we sure we need it? close it?

	p.exitNext()
	return program, err
}
