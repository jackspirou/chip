package parser

import (
	"fmt"

	"github.com/jackspirou/chip/src/chip/node"
	"github.com/jackspirou/chip/src/chip/ssa"
	"github.com/jackspirou/chip/src/chip/token"
	"github.com/jackspirou/chip/src/chip/types"
)

// Next Procedure Signature. Parse a procedure signature.
func (p *Parser) nextProcedureSignature() {
	p.enter()
	p.nextExpected(token.FUNC)
	procName := p.lit
	p.nextExpected(token.IDENT) // skip proc name
	p.nextExpected(token.LPAREN)
	proc := types.NewProcedureType()
	if p.tok != token.RPAREN {
		paramName := p.lit
		p.nextExpected(token.IDENT) // skip param name
		paramType := p.nextType()

		proc.InsertParam(paramType)
		reg := p.alloc.Request()
		paramDes := node.NewRegNode(paramType, reg)
		p.scope.Insert(paramName, paramDes)

		for p.tok == token.COMMA {
			p.next() // skip ','
			paramName = p.lit
			p.nextExpected(token.IDENT) // skip proc name
			paramType = p.nextType()

			proc.InsertParam(paramType)
			reg = p.alloc.Request()
			paramDes = node.NewRegNode(paramType, reg)
			p.scope.Insert(paramName, paramDes)
		}
	}
	p.nextExpected(token.RPAREN)
	if p.tok != token.LBRACE {
		valueType := p.nextType()
		proc.InsertValue(valueType)
		for p.tok == token.COMMA {
			p.next() // skip ','
			valueType = p.nextType()
			proc.InsertValue(valueType)
		}
	}
	label := ssa.NewLabel(procName)
	des := node.NewLabelNode(proc, label)
	_, err := p.scope.Global(procName, des)
	if err != nil {
		panic(err)
	}
	fmt.Println(label.String())
	p.exit()
}
