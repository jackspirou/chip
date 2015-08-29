package parser

import (
	"fmt"

	"github.com/jackspirou/chip/ssa"

	"github.com/jackspirou/chip/parser/token"
)

// nextProduct parses a product.
func (p *Parser) nextProduct() ssa.Node {
	p.enter()

	leftRegNode := p.nextTerm()

	for p.tok.Type == token.MUL || p.tok.Type == token.QUO || p.tok.Type == token.REM {

		// check(left, intType);

		tok := p.tok

		p.next() // skip '*' or '/'

		rightRegNode := p.nextTerm()
		fmt.Println(rightRegNode)

		// check(right, intType);
		// assembler.emit("# Product.");

		if tok.Type == token.MUL {
			//	assembler.emit("mul", left.getRegister(), left.getRegister(), right.getRegister());
		} else {
			// assembler.emit("div", left.getRegister(), left.getRegister(), right.getRegister());
		}

		// allocator.release(right.getRegister());
	}

	p.exit()
	return leftRegNode
}
