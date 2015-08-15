package parser

import "github.com/jackspirou/chip/src/chip/token"

// Next Package. Parse a package name.
func (p *Parser) nextPackage() {
	p.enter()
	p.nextExpected(token.PACKAGE)
	//p.tok.Type == // get package name
	p.nextExpected(token.IDENT)
	p.exit()
}
