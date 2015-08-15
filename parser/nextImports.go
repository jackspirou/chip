package parser

import "github.com/jackspirou/chip/token"

// Next Imports. Parse all package imports.
func (p *Parser) nextImports() {
	p.enter()
	p.nextExpected(token.IMPORT)
	if p.tok.Type == token.LPAREN {
		p.next() // skip '('
		for p.tok.Type != token.RPAREN {
			if p.tok.Type == token.IDENT {
				//p.tok.Type == // get package alias
				p.next()
				//p.tok.Type == // get package src
				p.nextExpected(token.STRING)
			} else {
				//p.tok.Type == // get package src
				p.nextExpected(token.STRING)
			}
		}
		p.nextExpected(token.RPAREN)
	} else {
		if p.tok.Type == token.IDENT {
			//p.tok.Type == // get package alias
			p.next()
			//p.tok.Type == // get package src
			p.nextExpected(token.STRING)
		} else {
			//p.tok.Type == // get package src
			p.nextExpected(token.STRING)
		}
	}
	p.exit()
}
