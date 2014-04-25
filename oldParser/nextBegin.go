package oldParser

import (
	"jak/common"
)

// Next Begin. Parse a begin statement.
func nextBegin() {
	common.Enter("nextBegin")
	scan.NextToken()
	if scan.GetToken() != common.CloseCurlyBracketToken {
		nextStatement()
		for scan.GetToken() == common.SemicolonToken {
			scan.NextToken()
			nextStatement()
		}
	}
	nextExpected(common.CloseCurlyBracketToken)
	common.Exit("nextBegin")
}
