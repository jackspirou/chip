package parser

import "github.com/jackspirou/chip/token"

// Next Declaration. Parse a declaration.
func (p *Parser) nextDeclaration() {
  p.enter()
  if p.tok == token.CONST {
    p.nextConstant()
  }else if p.tok == token.IDENT {
    p.token.String() // var name
    p.nextExpected(token.DEFINE)
    switch p.tok {
      case token.IDENT:
        p.token.String() //
    }
  }else{
    
  }



	var someType typ.Atype
	switch scan.GetToken() {
	case common.BoldIntToken:
		scan.NextToken() // Skip 'int'
		someType = intType
	case common.BoldStringToken:
		scan.NextToken()
		someType = stringType
	case common.OpenBracketToken:
		scan.NextToken() // Skip '['
		if scan.GetToken() == common.CloseBracketToken {
			scan.NextToken() // Skip ']'

		} else if scan.GetToken() == common.IntConstantToken {
			scan.NextToken()
			nextExpected(common.CloseBracketToken)
		} else {
			scan.Source.Error("Vector or defined array expected: ']' or 'integer constant'.")
		}

		if scan.GetToken() == common.BoldIntToken {
			someType = typ.NewArrayType(intType)
			scan.NextToken()
		} else if scan.GetToken() == common.BoldStringToken {
			someType = typ.NewArrayType(stringType)
			scan.NextToken()
		} else {
			scan.Source.Error("Type 'int' or 'string' expected.")
		}

	default:
		scan.Source.Error("Declaration expected, not:" + common.TokenToString(scan.GetToken()))
	}

	name := scan.GetString()

	nextExpected(common.NameToken)
	label := generator.NewLabel(name)
	des := NewLabelDescriptor(someType, label)
	symboltable.SetDescriptor(name, des)

	// CHeck For Initalization
	if scan.GetToken() == common.EqualToken {
		scan.NextToken()

		// Value
		scan.NextToken()
	}

	common.Exit("nextDeclaration")
}
