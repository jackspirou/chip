package parser

import (
	"jak/common"
)

//
// comparison	-------
// 		------->| sum |--->|-------------->---------------|-----> ???
//				-------	   |    						  |
//				     	   |    ------         -------    |
//						   |--->| <  |----|--->| sum |--->|
//						   |	------    |    -------
//						   |			  |
//						   |    ------    |
//						   |--->| <= |--->|
//						   |	------    |
//						   |			  |
//						   |    ------    |
//						   |--->| <> |--->|
//						   |	------    |
//						   |              |
//						   |    ------    |
//						   |--->| =  |--->|
//						   |	------    |
//						   |  			  |
//						   |    ------    |
//						   |--->| >  |--->|
//						   |	------    |
//						   |  			  |
//						   |    ------    |
//						   |--->| >= |--->|
//						        ------
//

// Next Comparison. Parse a comparison.
func nextComparison() *RegisterDescriptor {
	common.Enter("nextComparison")
	firstDes := nextSum()
	if isComparisonOperator() {
		// checkIntType(firstDes);
		for isComparisonOperator() {
			token := scan.GetToken()
			scan.NextToken() // Skip
			secondDes := nextSum()
			// checkIntType(secondDes);
			generate.Emit("# Next Comparison")

			switch token {
			case common.LessToken:
				generate.Branch(common.OP_SLT, firstDes.GetRegister().String(), firstDes.GetRegister().String(), secondDes.GetRegister().String())
			case common.LessEqualToken:
				generate.Branch(common.OP_SLE, firstDes.GetRegister().String(), firstDes.GetRegister().String(), secondDes.GetRegister().String())
			case common.LessGreaterToken:
				generate.Branch(common.OP_SNE, firstDes.GetRegister().String(), firstDes.GetRegister().String(), secondDes.GetRegister().String())
			case common.EqualEqualToken:
				generate.Branch(common.OP_SEQ, firstDes.GetRegister().String(), firstDes.GetRegister().String(), secondDes.GetRegister().String())
			case common.GreaterToken:
				generate.Branch(common.OP_SGT, firstDes.GetRegister().String(), firstDes.GetRegister().String(), secondDes.GetRegister().String())
			case common.GreaterEqualToken:
				generate.Branch(common.OP_SGE, firstDes.GetRegister().String(), firstDes.GetRegister().String(), secondDes.GetRegister().String())
			default:
				msg := "Snarl has an error in nextComparison. It found a " + common.TokenToString(token)
				scan.Source.Error(msg)
			}
			allocate.Release(secondDes.GetRegister())
		}
	}
	common.Exit("nextComparison")
	return firstDes
}
