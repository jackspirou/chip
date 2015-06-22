package ssa

// Opcode represents a bytecode operation.
type Opcode int

const (

	// variable declaration
	DeclareLong Opcode = iota
	DeclareDouble
	DeclareBoolean
	DeclareString
	DeclareReference
	DeclareArray
	DeclareStruct

	// type conversion
	LongToDouble
	DoubleToLong
	BooleanToString
	LongToString
	DoubleToString

	// unindexed copy
	AssignLong
	AssignDouble
	AssignBoolean
	AssignString

	// array indexed copy
	ArrayGetLong
	ArrayGetDouble
	ArrayGetBoolean
	ArrayGetString
	ArrayGetReference
	ArraySetLong
	ArraySetDouble
	ArraySetBoolean
	ArraySetString
	ArraySetReference

	// struct indexed copy
	StructGetLong
	StructGetDouble
	StructGetBoolean
	StructGetString
	StructGetReference
	StructSetLong
	StructSetDouble
	StructSetBoolean
	StructSetString
	StructSetReference

	// arithmetic
	AddLong
	AddFloat
	SubLong
	SubFloat
	MulLong
	MulFloat
	DivLong
	DivFloat
	NotBoolean
	OrBoolean
	AndBoolean
	ConcatString

	// compare
	CompareLongE
	CompareLongG
	CompareLongL
	CompareLongGE
	CompareLongLE
	CompareFloatE
	CompareFloatG
	CompareFloatL
	CompareFloatGE
	CompareFloatLE

	// control flow
	RETURN
	LABEL
	BRANCH

	// I/0
	PrintString
)
