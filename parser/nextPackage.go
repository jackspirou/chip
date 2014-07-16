package parser

import "github.com/JackSpirou/chip/token"

// Next Package. Parse a package name.
func (p *Parser) nextPackage() {
	p.enter()
	p.nextExpected(token.PACKAGE)
	//p.lit // get package name
	p.nextExpected(token.IDENT)
	p.exit()
}
