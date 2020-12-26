package parser

import (
	"github.com/jackspirou/chip/ast"
	"github.com/jackspirou/chip/token"
)

// nextTerm parses a term.
func (p *Parser) nextTerm() (term ast.Node, err error) {
	p.enterNext()

	if p.tok.Type == token.SUB || p.tok.Type == token.NOT {

		tok := p.tok

		p.next() // skip '-' (negative number) or '!'

		term, err = p.nextTerm()
		if err != nil {
			return nil, err
		}

		// // type check
		// if term.Object().Type() != token.INT {
		// 	panic("was not an integer")
		// }

		if tok.Type == token.NOT {
			// return a term node
			// assembler.emit("seq", descriptor.getRegister(), allocator.zero, descriptor.getRegister());
		} else {
			// assembler.emit("sub", descriptor.getRegister(), allocator.zero, descriptor.getRegister());
		}

	} else {
		// return a unit node
		/* des = */
		unit, err := p.nextUnit()
		if err != nil {
			return nil, err
		}
		term = unit
	}

	p.exitNext()

	return term, err
}
