package parser

import (
	"jak/common"
	"jak/generator"
)

//
// conjunction	--------------
//      ------->| comparison |--->|---------------->----------------|--->
//				      --------------	  |                                 |
//                                |    -------    --------------    |
//                                |--->| and |--->| comparison |--->|
//                                |    -------	  --------------    |
//                                |                                 |
//                                |----------------<----------------|
//

// Next Conjunction. Parse a conjunction.
func nextConjunction() *RegisterDescriptor {
	common.Enter("nextConjunction")
	des := nextComparison()
	if scan.GetToken() == common.BoldAndToken {
		scan.NextToken() // Skip the 'and'.
		reg := allocate.Request()
		label := generator.NewLabel("conjunction")
		generate.Emit("# Next Conjunction")
		generate.Branch(common.OP_SNE, reg.String(), des.GetRegister().String(), allocate.Zero.String())
		allocate.Release(des.GetRegister())
		generate.Imm(common.OP_BNE, allocate.Zero.String(), label.String())
		for scan.GetToken() == common.BoldAndToken {
			scan.NextToken() // Skip and token.
			des = nextComparison()
			// checkIntType(des)
			generate.Emit("# Next Conjunction")
			generate.Branch(common.OP_SNE, reg.String(), des.GetRegister().String(), allocate.Zero.String())
			allocate.Release(des.GetRegister())
			generate.Branch(common.OP_BEQ, reg.String(), allocate.Zero.String(), label.String())
		}
		generate.Label(label)
		des = NewRegisterDescriptor(des.GetType(), reg)
	}
	common.Exit("nextConjunction")
	return des
}
