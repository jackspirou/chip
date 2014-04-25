package oldParser

import (
	"jak/common"
	"jak/generator"
)

//
// expression	---------------
// 		------->| conjunction |--->|---------------->----------------|---> ???
//				---------------	   |                                 |
//								   |	------	 ---------------     |
//								   |--->| or |--->| conjunction |--->|
//								   |    ------	 ---------------     |
//								   |								 |
//								   |----------------<----------------|
//

// Next Expression. Parse an expression.
func nextExpression() *RegisterDescriptor {
	common.Enter("nextExpression")
	des := nextConjunction()
	if scan.GetToken() == common.BoldOrToken {
		scan.NextToken()
		reg := allocate.Request()
		label := generator.NewLabel("expression")
		generate.Emit("# Next Expression")
		generate.Branch(common.OP_SNE, reg.String(), des.GetRegister().String(), allocate.Zero.String())
		allocate.Release(des.GetRegister())
		generate.Branch(common.OP_BNE, reg.String(), allocate.Zero.String(), label.String())
		for scan.GetToken() == common.BoldOrToken {
			scan.NextToken()
			des = nextConjunction()
			// checkIntType(des)
			generate.Emit("# Next Expression")
			generate.Branch(common.OP_BNE, reg.String(), des.GetRegister().String(), allocate.Zero.String())
			allocate.Release(des.GetRegister())
			generate.Branch(common.OP_BNE, reg.String(), allocate.Zero.String(), label.String())
		}
		generate.Label(label)
		des = NewRegisterDescriptor(intType, reg)
	}
	common.Exit("nextExpression")
	return des
}
