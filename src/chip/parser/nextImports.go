package parser

import "github.com/jackspirou/chip/src/chip/token"

// Next Imports. Parse all package imports.
func (p *Parser) nextImports() {
	p.enter()
	p.nextExpected(token.IMPORT)
	if p.tok == token.LPAREN {
		p.next() // skip '('
		for p.tok != token.RPAREN {
			if p.tok == token.IDENT {
				//p.lit // get package alias
				p.next()
				//p.lit // get package src
				p.nextExpected(token.STRING)
			} else {
				//p.lit // get package src
				p.nextExpected(token.STRING)
			}
		}
		p.nextExpected(token.RPAREN)
	} else {
		if p.tok == token.IDENT {
			//p.lit // get package alias
			p.next()
			//p.lit // get package src
			p.nextExpected(token.STRING)
		} else {
			//p.lit // get package src
			p.nextExpected(token.STRING)
		}
	}
	p.exit()
}
