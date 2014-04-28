package parser

import "github.com/jackspirou/chip/token"

// Next Program. Parse the file (program) body.
func (p *Parser) nextProgram() {
  p.enter()
  for p.tok != token.EOF {
    if p.tok == token.FUNC {
      p.nextProc()
    }else{
      p.nextDec()
    }
  }
  p.exit()
}
