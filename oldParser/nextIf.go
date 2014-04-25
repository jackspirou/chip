package oldParser

import (
	"jak/common"
)

// Next If. Parse an if statement.
func nextIf() {
	common.Enter("nextIf")
	// (NEEDS TO BE FIXED)  ifLabel := generator.NewLabel("if")
	// (NEEDS TO BE FIXED) label := ifLabel
	for true {
		scan.NextToken()
		nextExpression()
		nextExpected(common.OpenCurlyBracketToken)
		nextStatement()
		for scan.GetToken() != common.CloseCurlyBracketToken {
			nextStatement()
		}
		nextExpected(common.CloseCurlyBracketToken)
		if scan.GetToken() == common.BoldElseToken {
			scan.NextToken()
			if scan.GetToken() == common.BoldIfToken {

			} else {
				nextExpected(common.OpenCurlyBracketToken)
				nextStatement()
				for scan.GetToken() != common.CloseCurlyBracketToken {
					nextStatement()
				}
				nextExpected(common.CloseCurlyBracketToken)
				break
			}
		}
	}
	common.Exit("nextIf")
}
