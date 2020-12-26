package parser

import (
	"github.com/jackspirou/chip/token"
	"github.com/jackspirou/chip/types"
)

// nextFunctionSignature parses a function signature.
func (p *Parser) nextFunctionSignature() {
	p.enterNext()

	// skip 'func'
	p.nextExpected(token.FUNC)

	// save the func name token
	//funcNameTok := p.tok

	p.nextExpected(token.IDENT)  // skip func name
	p.nextExpected(token.LPAREN) // skip '('

	// create a new func signature type
	funcSigType := types.NewFunc()

	// look for func parameters
	if p.tok.Type != token.RPAREN {

		// save the parameter name token
		// paramNameTok := p.tok

		// skip param name
		p.nextExpected(token.IDENT)

		// get the parameter type
		paramType := p.nextType()

		// add the parameter type to the func signature type
		funcSigType.AddParam(paramType)

		// allocate a register and create a new register node
		// reg := p.alloc.Request()
		// regNode := ssa.NewRegNode(paramType, reg)

		// add the parameter name token and register node to the current scope
		// p.scope.Add(paramNameTok, regNode)

		// look for more parameters
		for p.tok.Type == token.COMMA {
			p.next() // skip ','

			// save the parameter name token
			// paramNameTok = p.tok

			// skip parameter name
			p.nextExpected(token.IDENT)

			// get the parameter type
			paramType = p.nextType()

			// add the parameter type to the func signature type
			funcSigType.AddParam(paramType)

			// allocate a register and create a new register node
			// reg = p.alloc.Request()
			// regNode = ssa.NewRegNode(paramType, reg)

			// add the parameter name token and register node to the current scope
			// p.scope.Add(paramNameTok, regNode)
		}
	}

	// skip ')'
	p.nextExpected(token.RPAREN)

	// look for return value
	if p.tok.Type != token.LBRACE {

		// get the return value type
		paramType := p.nextType()

		// add the return value type to the func signature type
		funcSigType.AddValue(paramType)

		// look for more return types
		for p.tok.Type == token.COMMA {
			p.next() // skip ','

			// get the return value type
			paramType = p.nextType()

			// add the return value type to the func signature type
			funcSigType.AddValue(paramType)
		}
	}

	// create a func signature label and node
	// label := ssa.NewLabel(funcNameTok.String())
	// funcSigNode := ssa.NewFuncNode(funcSigType, label)

	// set the func signature node to the global scope and check for user error
	// if err := p.scope.Global(funcNameTok, funcSigNode); err != nil {
	// userErr(err, funcNameTok)
	// }

	p.exitNext()
}
