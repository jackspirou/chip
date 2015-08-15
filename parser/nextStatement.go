package parser

import (
	"log"

	"github.com/jackspirou/chip/token"
)

// nextStatement parse a statement.
func (p *Parser) nextStatement() {
	p.enter()
	switch p.tok.Type {
	case token.IDENT:
		// p.tok.String() // var or proc name
		p.next()
		for p.tok.Type == token.PERIOD {
			p.next() // skip '.'
			// p.tok.String() // var or proc name
			p.nextExpected(token.IDENT)
		}
		if p.tok.Type.Assignment() {
			p.nextAssignment()
		} else {
			switch p.tok.Type {
			case token.DEFINE:
				p.nextDeclaration()
			case token.LBRACK:
				p.next() // skip '['
				p.nextExpression()
				p.nextExpected(token.RBRACK)
				p.nextAssignment()
			case token.LPAREN:
				p.next() // skip '('
				if p.tok.Type != token.RPAREN {
					p.nextExpression()
					for p.tok.Type == token.COMMA {
						p.next() // skip ','
						p.nextExpression()
					}
				}
				p.nextExpected(token.RPAREN)
			default:
				log.Fatalf("assignment or declaration statement expected, got '%s'", p.tok.String())
			}
		}
	case token.IF:
		p.nextIf()
	/*
		case token.FOR:
			p.nextFor()
	*/
	case token.RETURN:
		p.nextReturn()
	/*
		case token.SWITCH:
			p.nextSwitch()
	*/
	case token.CONST:
		p.nextConstant()
	default:
		log.Fatalf("statement expected, got '%s'", p.tok.String())
	}
	p.exit()
}