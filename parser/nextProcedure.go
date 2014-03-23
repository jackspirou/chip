package parser

import (
	"jak/common"
	"jak/generator"
	"jak/typ"
)

// Next Procedure. Parse a procedure.
func nextProcedure() {
	common.Enter("nextProcedure")
	symboltable.Push()
	nextProcedureSignature()
	nextProcedureBody()
	symboltable.Pop()
	common.Exit("nextProcedure")
}

// Next Procedure Signature. Parse a procedure signature.
func nextProcedureSignature() {
	common.Enter("nextProcedureSignature")
	nextExpected(common.BoldFuncToken)
	name := scan.GetString()
	// global.Declare(name)
	nextExpected(common.NameToken)
	nextExpected(common.OpenParenToken)
	typ := typ.NewProcedureType()
	if scan.GetToken() != common.CloseParenToken {
		parTyp := nextType()
		typ.AddParameter(parTyp)
		parName := scan.GetString()
		nextExpected(common.NameToken)
		reg := allocate.Request()
		parDes := NewRegisterDescriptor(parTyp, reg)
		symboltable.SetDescriptor(parName, parDes)
		for scan.GetToken() == common.CommaToken {
			scan.NextToken()
			parTyp = nextType()
			typ.AddParameter(parTyp)
			parName = scan.GetString()
			nextExpected(common.NameToken)
			reg = allocate.Request()
			parDes = NewRegisterDescriptor(parTyp, reg)
			symboltable.SetDescriptor(parName, parDes)
		}
	}
	nextExpected(common.CloseParenToken)
	typ.AddValue(nextType())
	label := generator.NewLabel(name)
	des := NewLabelDescriptor(typ, label)
	err := symboltable.SetGlobalDescriptor(name, des)
	check(err)
	generate.Label(label)
	common.Exit("nextProcedureSignature")
}

// Next Procedure Body. Parse the next procedure body.
func nextProcedureBody() {
	common.Enter("nextProcedureBody")
	nextExpected(common.OpenCurlyBracketToken)
	nextStatement()
	for scan.GetToken() != common.CloseCurlyBracketToken {
		nextStatement()
	}
	nextExpected(common.CloseCurlyBracketToken)
	common.Exit("nextProcedureBody")
}
