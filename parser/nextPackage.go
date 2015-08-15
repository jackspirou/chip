package parser

import "github.com/jackspirou/chip/token"

// nextPackage parses a package name.
func (p *Parser) nextPackage() {
	p.enter()
	p.nextExpected(token.PACKAGE)
	p.nextExpected(token.IDENT)
	p.exit()
}
