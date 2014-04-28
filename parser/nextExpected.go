package parser

import (
  "fmt"
  "github.com/jackspirou/chip/support"
  "github.com/jackspirou/chip/token"
)


// Next Expected. Expects argument as the next token.
func (p *Parser) nextExpected(expected token.Tokint) {
  if p.tok == expected {
    p.next()
  } else {
    msg := "\"" + expected.String() + "\" expected instead of \"" + p.token.String() + "\"."
    panic(msg)
  }
}
