package parser

import "github.com/jackspirou/chip/token"

// nextPackage parses a package name.
func (p *Parser) nextPackage() {
	p.enterNext()

	if p.tok.Type == token.PACKAGE {
		p.next()
		p.nextExpected(token.IDENT)
	}

	p.exitNext()
}
