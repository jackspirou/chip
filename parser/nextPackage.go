package parser

import "github.com/jackspirou/chip/token"

// nextPackage parses a package name.
func (p *Parser) nextPackage() {
	p.enter()
	if p.tok.Type == token.PACKAGE {
		p.nextExpected(token.IDENT)
	}
	p.exit()
}
