package parser

import (
	"fmt"

	"github.com/jackspirou/chip/ssa"

	"github.com/jackspirou/chip/parser/token"
)

// nextComparison parses a comparison.
func (p *Parser) nextComparison() ssa.Node {
	p.enter()

	leftRegNode := p.nextSum()

	if tok := p.tok; tok.Type.Comparison() {

		// check(left, intType);

		p.next() // comparison

		/*
			fmt.Println(p.tok)
			rightRegNode := p.nextSum()
			fmt.Println(p.tok)

			fmt.Println(rightRegNode)
			// check(right, intType);
			// assembler.emit("# Comparison.");
		*/

		switch tok.Type {
		case token.EQL: // left == right

			fmt.Println(p.tok)
			fmt.Println("parsing == ")
			p.next() // ==
			fmt.Println(p.tok)

		case token.LSS: // left < right
			// assembler.emit("slt", left.getRegister(), left.getRegister(), right.getRegister())
		case token.GTR: // left > right
			// assembler.emit("sgt", right.getRegister(), right.getRegister(), left.getRegister());
		case token.NEQ: // left != right
			p.next() // !=
		case token.LEQ: // left <= right
			// assembler.emit("sle", left.getRegister(), left.getRegister(), right.getRegister());
		case token.GEQ: // left >= right
			// assembler.emit("sge", left.getRegister(), left.getRegister(), right.getRegister());
		}
		// allocator.release(right.getRegister());
	}

	p.exit()
	return leftRegNode
}
