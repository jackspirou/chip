package oldParser

import (
	"jak/common"
)

// Next Expected. Expects argument as the next token.
func nextExpected(expectedToken int) {
	if scan.GetToken() == expectedToken {
		scan.NextToken()
	} else {
		msg := "\"" + common.TokenToString(expectedToken) +
			"\" expected instead of \"" +
			common.TokenToString(scan.GetToken()) + "\"."
		scan.Source.Error(msg)
	}
}
