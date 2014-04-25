package oldParser

import (
	"jak/common"
)

//
// Supporting Methods.
//

// Is Comparison Operator. Check if a token is a comparison token.
func isComparisonOperator() bool {
	common.Enter("isComparisonOperator")
	switch scan.GetToken() {
	case common.LessToken, common.LessEqualToken, common.LessGreaterToken, common.EqualEqualToken, common.GreaterToken, common.GreaterEqualToken:
		common.Exit(common.TokenToString(scan.GetToken()) + " isComparisonOperator => true")
		return true
	default:
		common.Exit(common.TokenToString(scan.GetToken()) + " isComparisonOperator => false")
		return false
	}
}

// Is Sum Operator. Check if a token is a sum token.
func isSumOperator() bool {
	return scan.GetToken() == common.PlusToken || scan.GetToken() == common.DashToken
}

// Is Product Operator. Check if a token is a product operator token.
func isProductOperator() bool {
	return scan.GetToken() == common.StarToken || scan.GetToken() == common.SlashToken
}

// Test to see if the term is a '-' or a 'not' for nextTerm.
func isTermOperator() bool {
	return scan.GetToken() == common.DashToken || scan.GetToken() == common.BoldNotToken
}

// Test to see if the term is an 'int', a 'string', or a '[' for nextDeclaration.
func isDeclaration() bool {
	return scan.GetToken() == common.BoldIntToken || scan.GetToken() == common.BoldStringToken || scan.GetToken() == common.OpenBracketToken
}

func check(err error) {
	if err != nil {
		scan.Source.Error(err.Error())
	}
}
