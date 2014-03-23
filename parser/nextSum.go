package parser

import (
	"jak/common"
)

// Next Sum. Parse a sum.
func nextSum() *RegisterDescriptor {
	common.Enter("nextSum")
	firstDes := nextProduct()
	if isSumOperator() {
		// checkIntType(firstDes);
		for isSumOperator() {
			token := scan.GetToken()
			scan.NextToken() // Skip the '+' or '-'.
			secondDes := nextProduct()
			// checkIntType(secondDes);
			generate.Emit("# Next Sum")
			switch token {
			case common.PlusToken:
				generate.Branch(common.OP_ADD, firstDes.GetRegister().String(), firstDes.GetRegister().String(), secondDes.GetRegister().String())
			case common.DashToken:
				generate.Branch(common.OP_SUB, firstDes.GetRegister().String(), firstDes.GetRegister().String(), secondDes.GetRegister().String())
			default:
				panic("Snarl has an error in nextSum.")
			}
			allocate.Release(secondDes.GetRegister())
		}
	}
	common.Exit("nextSum")
	return firstDes
}
