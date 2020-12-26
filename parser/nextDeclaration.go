package parser

import (
	"github.com/jackspirou/chip/ast"
	"github.com/jackspirou/chip/token"
)

// nextDeclaration parses a declaration.
func (p *Parser) nextDeclaration() (declaration ast.Node, err error) {
	p.enterNext()

	p.nextExpected(token.DEFINE)

	p.nextExpression()

	p.exitNext()
	return declaration, err
}

// Test if the current token is a "CONST", a 'string', or a '[' for nextDeclaration.
/*
func (p *Parser) isDeclaration() bool {
	return p.tok.Type == token. || scan.GetToken() == common.BoldStringToken || scan.GetToken() == common.OpenBracketToken
}
*/
