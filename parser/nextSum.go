package parser

import (
	"github.com/jackspirou/chip/ssa"

	"github.com/jackspirou/chip/token"
)

// nextSum parses a sum.
func (p *Parser) nextSum() ssa.Node {
	p.enter()

	p.nextProduct()

	for p.tok.Type == token.ADD || p.tok.Type == token.SUB {

		// check(left, intType);

		tok := p.tok
		p.next() // skip '+' or '-'

		p.nextProduct()

		// check(right, intType);

		if tok.Type == token.ADD {
			// assembler.emit("add", left.getRegister(), left.getRegister(), right.getRegister());
		} else {
			// assembler.emit("sub", left.getRegister(), left.getRegister(), right.getRegister());
		}
		// allocator.release(right.getRegister());
	}

	p.exit()
	return nil
}
