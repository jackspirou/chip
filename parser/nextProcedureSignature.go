package parser

import (
	"github.com/jackspirou/chip/node"
	"github.com/jackspirou/chip/ssa"
	"github.com/jackspirou/chip/token"
	"github.com/jackspirou/chip/types"
)

// Next Procedure Signature. Parse a procedure signature.
func (p *Parser) nextProcedureSignature() {
	p.enter()
	p.nextExpected(token.FUNC)
	procName := p.tok.String()
	p.nextExpected(token.IDENT) // skip proc name
	p.nextExpected(token.LPAREN)
	proc := types.NewProc()
	if p.tok.Type != token.RPAREN {
		paramName := p.tok.String()
		p.nextExpected(token.IDENT) // skip param name
		paramType := p.nextType()

		proc.AddParam(paramType)
		reg := p.alloc.Request()
		paramDes := node.NewReg(paramType, reg)
		p.scope.Add(paramName, paramDes)

		for p.tok.Type == token.COMMA {
			p.next() // skip ','
			paramName = p.tok.String()
			p.nextExpected(token.IDENT) // skip proc name
			paramType = p.nextType()

			proc.AddParam(paramType)
			reg = p.alloc.Request()
			paramDes = node.NewReg(paramType, reg)
			p.scope.Add(paramName, paramDes)
		}
	}
	p.nextExpected(token.RPAREN)
	if p.tok.Type != token.LBRACE {
		valueType := p.nextType()
		proc.AddValue(valueType)
		for p.tok.Type == token.COMMA {
			p.next() // skip ','
			valueType = p.nextType()
			proc.AddValue(valueType)
		}
	}
	label := ssa.NewLabel(procName)
	des := node.NewLabel(proc, label)
	_, err := p.scope.Global(procName, des)
	if err != nil {
		panic(err)
	}
	p.exit()
}
