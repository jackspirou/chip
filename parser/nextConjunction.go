package parser

import (
	"fmt"

	"github.com/jackspirou/chip/parser/token"
	"github.com/jackspirou/chip/ssa"
)

// nextConjunction parses a conjunction.
func (p *Parser) nextConjunction() ssa.Node {
	p.enter()

	fmt.Println(p.tok)
	leftRegNode := p.nextComparison()
	fmt.Println(p.tok)

	label := ssa.NewLabel("conjunction")
	fmt.Println(label)

	found := false
	if p.tok.Type == token.LOR {
		found = true
		// assembler.emit("sne", leftRegNode.Register(), alloc.Zero, label)
	}

	for p.tok.Type == token.LAND {

		// check(leftRegNode, intType)

		p.next() // skip '&&'

		des := p.nextComparison()
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
