package parser

import "github.com/jackspirou/chip/token"

// Next Statement. Parse a statement.
func (p *Parser) nextStatement() {
	p.enter()
	switch p.tok {
	case token.IDENT:
		p.lit // var or proc name
		p.next()
    switch p.tok {
      case token.IsAssignment(p.tok):
        p.nextAssignment()
      case token.DEFINE:
        p.nextDeclaration()
      case token.LBRACK:
        p.next() // skip '['
        p.nextExpression()
        p.nextExpected(token.RBRACK)
        p.nextAssignment()
      case token.LPAREN:
        p.next() // skip '('
        if p.tok != token.RPAREN {
          p.nextExpression()
          for p.tok == token.COMMA {
            p.next() // skip ','
            p.nextExpression()
          }
        }
        p.nextExpected(token.RPAREN)
      default:
        panic("Expected an assignment or declaration.")
    }
	case token.IF:
    p.nextIf()
	case token.FOR:
    p.nextFor()
	case token.RETURN:
    p.nextReturn()
	case token.SWITCH:
    p.nextSwitch()
	case token.CONST:
    p.nextConstant()
  default:
    panic("Statement Expected.")
	}
	p.exit()
}
