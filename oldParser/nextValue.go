package oldParser

import (
	"jak/common"
)

// Next Value. Parse a value statement.
func nextValue() {
	common.Enter("nextValue")
	nextExpected(common.BoldValueToken)
	nextExpression()
	common.Exit("nextValue")
}
