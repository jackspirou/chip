package parser

import "github.com/jackspirou/chip/token"

// Next Procedure Signature. Parse a procedure signature.
func (p *Parser) nextProcedureSignature() {
  p.enter()
  p.nextExpected(token.FUNC)
  //p.lit // proc name
  p.nextExpected(token.IDENT)
  p.nextExpected(token.LPAREN)
  if p.tok != token.RPAREN {
    //p.lit // param name
    p.nextExpected(token.IDENT)
    //p.lit // param type
    p.nextExpected(token.IDENT)
    for p.tok == token.COMMA {
      p.next() // skip ','
      //p.lit // param name
      p.nextExpected(token.IDENT)
      //p.lit // param type
      p.nextExpected(token.IDENT)
    }
  }
  p.nextExpected(token.RPAREN)
  if p.tok != token.LBRACE {
    //p.lit // return type
    p.nextExpected(token.IDENT)
    for p.tok == token.COMMA {
      p.next() // skip ','
      //p.lit // return type
      p.nextExpected(token.IDENT)
    }
  }
  p.exit()
}
