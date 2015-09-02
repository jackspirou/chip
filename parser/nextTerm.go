package parser

import (
	"github.com/jackspirou/chip/parser/token"
	"github.com/jackspirou/chip/ssa"
)

// nextTerm parses a term.
func (p *Parser) nextTerm() ssa.Node {
	p.enter()

	var des ssa.Node

	if p.tok.Type == token.SUB || p.tok.Type == token.NOT {

		tok := p.tok

		p.next() // skip '-' or '!'

		des = p.nextTerm()
		//	check(descriptor, intType)

		if tok.Type == token.NOT {
			//	assembler.emit("seq", descriptor.getRegister(), allocator.zero, descriptor.getRegister());
		} else {
			//	assembler.emit("sub", descriptor.getRegister(), allocator.zero, descriptor.getRegister());
		}

	} else {
		des = p.nextUnit()
	}

	p.exit()
	return des
}
