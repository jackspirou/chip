package ssa

// TAC represents a three address code instruction.
type TAC struct {
	op  Opcode
	err error
}

func NewTAC() *TAC {
	return &TAC{}
}
