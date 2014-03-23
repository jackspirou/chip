package parser

import (
	"jak/common"
)

// Next Code. Parse a code statement.
func nextCode() {
	common.Enter("nextCode")
	scan.NextToken()
	mipsCode := scan.GetString()
	nextExpected(common.StringConstantToken)
	generate.Emit("# Static Code")
	generate.Emit(mipsCode)
	common.Exit("nextCode")
}
