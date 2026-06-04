// Package code defines chip's bytecode: opcodes, instructions, and the compiled
// program structures the VM executes.
package code

// Opcode is a chip VM instruction operation.
type Opcode uint8

const (
	OpConst        Opcode = iota // push Constants[operand]
	OpTrue                       // push true
	OpFalse                      // push false
	OpPop                        // discard top of stack
	OpAdd                        // push a + b
	OpSub                        // push a - b
	OpMul                        // push a * b
	OpDiv                        // push a / b
	OpRem                        // push a % b
	OpNeg                        // push -a
	OpNot                        // push !a
	OpEqual                      // push a == b
	OpNotEqual                   // push a != b
	OpLess                       // push a < b
	OpLessEqual                  // push a <= b
	OpGreater                    // push a > b
	OpGreaterEqual               // push a >= b
	OpGetLocal                   // push locals[operand]
	OpSetLocal                   // locals[operand] = pop
	OpJump                       // ip = operand
	OpJumpIfFalse                // pop; if false, ip = operand
	OpCall                       // call Functions[operand]
	OpReturn                     // return the value on top of stack
	OpReturnVoid                 // return no value
	OpPrint                      // print the top `operand` values
	OpMakeArray                  // build a slice from the top `operand` values
	OpIndex                      // push arr[i]
	OpSetIndex                   // arr[i] = v
	OpLen                        // push len(x)
)

var opNames = [...]string{
	OpConst:        "Const",
	OpTrue:         "True",
	OpFalse:        "False",
	OpPop:          "Pop",
	OpAdd:          "Add",
	OpSub:          "Sub",
	OpMul:          "Mul",
	OpDiv:          "Div",
	OpRem:          "Rem",
	OpNeg:          "Neg",
	OpNot:          "Not",
	OpEqual:        "Equal",
	OpNotEqual:     "NotEqual",
	OpLess:         "Less",
	OpLessEqual:    "LessEqual",
	OpGreater:      "Greater",
	OpGreaterEqual: "GreaterEqual",
	OpGetLocal:     "GetLocal",
	OpSetLocal:     "SetLocal",
	OpJump:         "Jump",
	OpJumpIfFalse:  "JumpIfFalse",
	OpCall:         "Call",
	OpReturn:       "Return",
	OpReturnVoid:   "ReturnVoid",
	OpPrint:        "Print",
	OpMakeArray:    "MakeArray",
	OpIndex:        "Index",
	OpSetIndex:     "SetIndex",
	OpLen:          "Len",
}

// String returns the opcode's mnemonic.
func (op Opcode) String() string {
	if int(op) < len(opNames) && opNames[op] != "" {
		return opNames[op]
	}
	return "Op(?)"
}

// hasOperand reports whether op uses its operand field (used by the
// disassembler).
func (op Opcode) hasOperand() bool {
	switch op {
	case OpConst, OpGetLocal, OpSetLocal, OpJump, OpJumpIfFalse, OpCall, OpPrint, OpMakeArray:
		return true
	}
	return false
}
