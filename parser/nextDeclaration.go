package parser

import "github.com/jackspirou/chip/token"

// Next Declaration. Parse a declaration.
func (p *Parser) nextDeclaration() {
	p.enter()
	p.nextExpected(token.DEFINE)
	p.nextExpression()
	p.exit()
}

// Test if the current token is a "CONST", a 'string', or a '[' for nextDeclaration.
/*
func (p *Parser) isDeclaration() bool {
	return p.tok.Type == token. || scan.GetToken() == common.BoldStringToken || scan.GetToken() == common.OpenBracketToken
}
*/
