package oldParser

import (
	"jak/common"
)

// Next Term. Parse a term.
func nextTerm() *RegisterDescriptor {
	common.Enter("nextTerm")
	des := &RegisterDescriptor{}
	if isTermOperator() {
		token := scan.GetToken()
		scan.NextToken() // Skip the '-' or 'not'.
		des = nextTerm()
		// checkIntType(des);
		generate.Emit("# Next Term")
		switch token {
		case common.DashToken:
			generate.Branch(common.OP_SUB, des.GetRegister().String(), allocate.Zero.String(), des.GetRegister().String())
		case common.BoldNotToken:
			generate.Branch(common.OP_SEQ, des.GetRegister().String(), allocate.Zero.String(), des.GetRegister().String())
		default:
			panic("Snarl has an error in nextTerm.")
		}
	} else {
		des = nextUnit()
	}
	common.Exit("nextTerm")
	return des
}
