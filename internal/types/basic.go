package types

// Kind identifies a basic type.
type Kind uint8

const (
	KindInvalid Kind = iota // a type error or unresolved type
	KindVoid                // the absence of a value
	KindInt
	KindFloat
	KindBool
	KindString
)

// Basic is a predeclared scalar type (or a sentinel such as invalid/void).
type Basic struct {
	Kind Kind
}

func (Basic) typ() {}

func (b Basic) String() string {
	switch b.Kind {
	case KindVoid:
		return "void"
	case KindInt:
		return "int"
	case KindFloat:
		return "float"
	case KindBool:
		return "bool"
	case KindString:
		return "string"
	default:
		return "invalid"
	}
}

// Predeclared basic types.
var (
	Invalid = Basic{KindInvalid}
	Void    = Basic{KindVoid}
	Int     = Basic{KindInt}
	Float   = Basic{KindFloat}
	Bool    = Basic{KindBool}
	String  = Basic{KindString}
)
