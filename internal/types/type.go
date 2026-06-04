// Package types defines chip's static type system.
package types

// Type is implemented by all chip types.
type Type interface {
	String() string
	typ()
}

// Identical reports whether a and b denote the same type.
func Identical(a, b Type) bool {
	switch a := a.(type) {
	case Basic:
		b, ok := b.(Basic)
		return ok && a.Kind == b.Kind
	case Slice:
		b, ok := b.(Slice)
		return ok && a.Elem != nil && b.Elem != nil && Identical(a.Elem, b.Elem)
	case *Signature:
		b, ok := b.(*Signature)
		if !ok || len(a.Params) != len(b.Params) {
			return false
		}
		for i := range a.Params {
			if !Identical(a.Params[i], b.Params[i]) {
				return false
			}
		}
		switch {
		case a.Result == nil && b.Result == nil:
			return true
		case a.Result == nil || b.Result == nil:
			return false
		default:
			return Identical(a.Result, b.Result)
		}
	}
	return false
}

// Lookup returns the predeclared basic type named name.
func Lookup(name string) (Basic, bool) {
	switch name {
	case "int":
		return Int, true
	case "float":
		return Float, true
	case "bool":
		return Bool, true
	case "string":
		return String, true
	}
	return Invalid, false
}
