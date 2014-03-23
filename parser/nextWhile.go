package parser

import (
	"jak/common"
)

// Next While. Parse a while statement.
func nextWhile() {
	common.Enter("nextWhile")
	scan.NextToken()
	nextExpression()
	nextExpected(common.OpenCurlyBracketToken)
	nextStatement()
	for scan.GetToken() != common.CloseCurlyBracketToken {
		nextStatement()
	}
	nextExpected(common.CloseCurlyBracketToken)
	common.Exit("nextWhile")
}
