package parser

import (
	"fmt"

	"github.com/jackspirou/chip/parser/token"
	"github.com/jackspirou/chip/ssa"
)

// nextExpression parses an expression.
func (p *Parser) nextExpression() ssa.Node {
	p.enter()

	fmt.Println(p.tok)
	leftRegNode := p.nextConjunction()
	fmt.Println(p.tok)

	label := ssa.NewLabel("expression")
	fmt.Println(label)

	found := false
	if p.tok.Type == token.LOR {
		found = true
		// assembler.emit("sne", leftRegNode.Register(), alloc.Zero, label)
	}

	for p.tok.Type == token.LOR {

		// check(leftRegNode, intType)

		p.next() // skip '||'

		des := p.nextConjunction()

		fmt.Println(des)
		// check(des, intType)

		// assembler.emit("sne", leftRegNode.Register(), des.Register(), alloc.Zero)

		// alloc.Release(des.Register())
	}

	if found {
		// assembler.emit(label)
	}

	p.exit()
	return leftRegNode
}
