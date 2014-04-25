package oldParser

import (
	"jak/common"
)

//
// sum			-----------
// 		------->| product |--->|---------------->--------------------------> ???
//				-----------	   |
//				     	  	   |    ------         -----------
//						  	   |--->| <  |--->|--->| product |--->|
//						  	   |	------    |    -----------	  |
//						  	   |	 	      |					  |
//						  	   |    ------    | 			      |
//						  	   |--->| <= |--->|  				  |
//						  	   |	------        			      |
//						  	   |				  				  |
//						  	   |----------------<-----------------|
//

// Next Product. Parse a product.
func nextProduct() *RegisterDescriptor {
	common.Enter("nextProduct")
	firstDes := nextTerm()
	if isProductOperator() {
		// checkIntType(firstDes);
		for isProductOperator() {
			token := scan.GetToken()
			scan.NextToken() // Skip the '*' or '/'.
			secondDes := nextTerm()
			// checkIntType(secondDes);
			switch token {
			case common.StarToken:
				generate.Branch(common.OP_MULT, firstDes.GetRegister().String(), firstDes.GetRegister().String(), secondDes.GetRegister().String())
			case common.SlashToken:
				generate.Branch(common.OP_DIV, firstDes.GetRegister().String(), firstDes.GetRegister().String(), secondDes.GetRegister().String())
			default:
				panic("Snarl has an error in nextProduct.")
			}
			allocate.Release(secondDes.GetRegister())
		}
	}
	common.Exit("nextProduct")
	return firstDes
}
