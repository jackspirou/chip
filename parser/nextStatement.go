package parser

import (
	// "fmt"
	"jak/common"
	"jak/generator"
	"jak/typ"
)

// Next Statement. Parse a statement.
func nextStatement() {
	common.Enter("nextStatement")
	if isDeclaration() {
		// nextLocalDeclaration()
		nextDeclaration()
	} else {
		// fmt.Println(common.TokenToString(scan.GetToken()))
		switch scan.GetToken() {
		case common.NameToken:
			name := scan.GetString()
			scan.NextToken() // Skip nameToken
			nameDes, err := symboltable.GetDescriptor(name)
			var procedureType *typ.ProcedureType
			if nameDes != nil {
				massie.ImplyType(nameDes.GetType())
			}
			if scan.GetToken() == common.EqualToken {
				check(err)
				scan.NextToken()
				regDes := nextExpression()
				allocate.Release(regDes.GetRegister())
			} else if scan.GetToken() == common.OpenBracketToken {
				check(err)
				scan.NextToken()
				regDes := nextExpression()
				allocate.Release(regDes.GetRegister())
				nextExpected(common.CloseBracketToken)
				nextExpected(common.EqualToken)
				regDes = nextExpression()
				allocate.Release(regDes.GetRegister())
			} else if scan.GetToken() == common.OpenParenToken {
				scan.NextToken()
				if nameDes == nil {
					procedureType = typ.NewProcedureType()
				}
				if scan.GetToken() != common.CloseParenToken {
					regDes := nextExpression()
					if nameDes == nil {
						procedureType.AddParameter(regDes.GetType())
					}
					allocate.Release(regDes.GetRegister())
					numOfArgs := 1
					for scan.GetToken() == common.CommaToken {
						scan.NextToken()
						regDes = nextExpression()
						if nameDes == nil {
							procedureType.AddParameter(regDes.GetType())
						}
						allocate.Release(regDes.GetRegister())
						numOfArgs++
					}
					nextExpected(common.CloseParenToken)
					if nameDes == nil {
						label := generator.NewLabel(name)
						des := NewLabelDescriptor(procedureType, label)
						massie.Trust(name, des)
						symboltable.SetGlobalDescriptor(name, des)
					}
				} else {
					nextExpected(common.CloseParenToken)
					if nameDes == nil {
						procedureType = typ.NewProcedureType()
						label := generator.NewLabel(name)
						des := NewLabelDescriptor(procedureType, label)
						massie.Trust(name, des)
						symboltable.SetGlobalDescriptor(name, des)
					}
				}
			} else {
				scan.Source.Error("Expected a '[', '(', or '='.")
			}
		case common.BoldCodeToken:
			nextCode()
		case common.BoldIfToken:
			nextIf()
		case common.BoldValueToken:
			nextValue()
		case common.BoldWhileToken:
			nextWhile()
		default:
			scan.Source.Error("Statement expected.")
		}
	}
	common.Exit("nextStatement")
}
