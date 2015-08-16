package parser

import (
	"github.com/jackspirou/chip/node"
	"github.com/jackspirou/chip/ssa"
	"github.com/jackspirou/chip/token"
	"github.com/jackspirou/chip/types"
)

// nextProcedureSignature parses a procedure signature.
func (p *Parser) nextProcedureSignature() {
	p.enter()

	p.nextExpected(token.FUNC)
	ProcNameTok := p.tok
	p.nextExpected(token.IDENT) // skip proc name
	p.nextExpected(token.LPAREN)
	proc := types.NewProc()
	if p.tok.Type != token.RPAREN {
		pname := p.tok.String()
		p.nextExpected(token.IDENT) // skip param name
		ptype := p.nextType()

		proc.AddParam(ptype)
		reg := p.alloc.Request()
		regNode := node.NewReg(ptype, reg)
		p.scope.Add(pname, regNode)

		for p.tok.Type == token.COMMA {
			p.next() // skip ','
			pname = p.tok.String()
			p.nextExpected(token.IDENT) // skip proc name
			ptype = p.nextType()

			proc.AddParam(ptype)
			reg = p.alloc.Request()
			regNode = node.NewReg(ptype, reg)
			p.scope.Add(pname, regNode)
		}
	}
	p.nextExpected(token.RPAREN)

	if p.tok.Type != token.LBRACE {
		ptype := p.nextType()
		proc.AddValue(ptype)
		for p.tok.Type == token.COMMA {
			p.next() // skip ','
			ptype = p.nextType()
			proc.AddValue(ptype)
		}
	}

	label := ssa.NewLabel(ProcNameTok)
	des := node.NewLabel(proc, label)

	if err := p.scope.Global(ProcNameTok, des); err != nil {
		userErr(err, ProcNameTok)
	}
	p.exit()
}
