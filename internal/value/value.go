// Package value defines the runtime values manipulated by the chip VM.
package value

import (
	"strconv"
	"strings"
)

// Kind identifies the dynamic type of a Value.
type Kind uint8

const (
	KindInt Kind = iota
	KindFloat
	KindBool
	KindString
	KindSlice
)

// Value is a chip runtime value: a small tagged union.
type Value struct {
	Kind Kind
	n    int64   // int, and bool as 0/1
	f    float64 // float
	s    string  // string
	a    []Value // slice elements (reference semantics)
}

// Int returns an integer value.
func Int(n int64) Value { return Value{Kind: KindInt, n: n} }

// Float returns a floating-point value.
func Float(f float64) Value { return Value{Kind: KindFloat, f: f} }

// Bool returns a boolean value.
func Bool(b bool) Value {
	v := Value{Kind: KindBool}
	if b {
		v.n = 1
	}
	return v
}

// Str returns a string value.
func Str(s string) Value { return Value{Kind: KindString, s: s} }

// Slice returns a slice value backed by elems.
func Slice(elems []Value) Value { return Value{Kind: KindSlice, a: elems} }

// AsInt returns the value as an int64.
func (v Value) AsInt() int64 { return v.n }

// AsFloat returns the value as a float64.
func (v Value) AsFloat() float64 { return v.f }

// AsBool returns the value as a bool.
func (v Value) AsBool() bool { return v.n != 0 }

// AsStr returns the value as a string.
func (v Value) AsStr() string { return v.s }

// AsSlice returns the value's backing slice.
func (v Value) AsSlice() []Value { return v.a }

// String formats the value for printing.
func (v Value) String() string {
	switch v.Kind {
	case KindInt:
		return strconv.FormatInt(v.n, 10)
	case KindFloat:
		return strconv.FormatFloat(v.f, 'g', -1, 64)
	case KindBool:
		if v.n != 0 {
			return "true"
		}
		return "false"
	case KindString:
		return v.s
	case KindSlice:
		parts := make([]string, len(v.a))
		for i, e := range v.a {
			parts[i] = e.String()
		}
		return "[" + strings.Join(parts, " ") + "]"
	default:
		return "<value>"
	}
}
