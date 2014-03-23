package parser

import (
	"jak/common"
	"jak/typ"
)

// Next Type.  Parse the next type token.
func nextType() typ.Atype {
	common.Enter("nextType")
	switch scan.GetToken() {
	case common.BoldIntToken:
		scan.NextToken()
		intType := typ.NewBasicType(common.BASIC_INT)
		common.Exit("nextType")
		return intType
	case common.BoldStringToken:
		scan.NextToken()
		stringType := typ.NewBasicType(common.BASIC_STRING)
		common.Exit("nextType")
		return stringType
	case common.OpenBracketToken:
		scan.NextToken()
		nextExpected(common.CloseBracketToken)
		someType := nextType()
		arrayType := typ.NewArrayType(someType)
		common.Exit("nextType")
		return arrayType
	default:
		scan.Source.Error("Type expected.")
	}
	panic("Satisfy Return")
}
