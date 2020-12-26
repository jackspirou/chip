package parser

import (
	"log"

	"github.com/jackspirou/chip/ast"
	"github.com/jackspirou/chip/token"
)

// nextStatement parse a statement.
func (p *Parser) nextStatement() (statement ast.Node, err error) {
	p.enterNext()

	switch p.tok.Type {
	case token.IDENT:
		p.next()
		for p.tok.Type == token.PERIOD {
			p.next() // '.'
			p.nextExpected(token.IDENT)
		}
		if p.tok.Type.Assignment() {
			p.nextAssignment()
		} else {
			switch p.tok.Type {
			case token.DEFINE:
				p.nextDeclaration()
			case token.LBRACK:
				p.next() // '['
				p.nextExpression()
				p.nextExpected(token.RBRACK)
				p.nextAssignment()
			case token.LPAREN:
				p.next() // '('
				if p.tok.Type != token.RPAREN {
					p.nextExpression()
					for p.tok.Type == token.COMMA {
						p.next() // ','
						p.nextExpression()
					}
				}
				p.nextExpected(token.RPAREN)
			default:
				log.Fatalf("assignment or declaration statement expected, got '%s'", p.tok)
			}
		}
	case token.IF:
		p.nextIf()
	case token.FOR:
		p.nextFor()
	case token.RETURN:
		p.nextReturn()
	/*
		case token.SWITCH:
			p.nextSwitch()
	*/
	default:
		log.Fatalf("statement expected, got '%s'", p.tok)
	}

	p.exitNext()
	return statement, err
}
