package parser

import "github.com/jackspirou/chip/token"

// Next Sum. Parse a sum.
func (p *Parser) nextSum() {
  p.enter()
  p.nextProduct()
  for p.tok == token.ADD || p.tok == token.SUB {
    p.next() // skip '+' or '-'
    p.nextProduct()
  }
  p.exit()
}
