// Package value defines the runtime values manipulated by the chip VM.
package value

import (
	"math"
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

// Value is a chip runtime value: a small tagged union. The int, bool, and float
// payloads share one 8-byte word (a float is held as its IEEE-754 bits), which
// keeps Value at 56 bytes — a scalar never needs the string header or the slice
// header, so overlapping the numeric payload costs nothing and shrinks every
// copy the tree-walker makes.
type Value struct {
	Kind Kind
	n    uint64  // int (as bits), bool (0/1), or float64 bits — by Kind
	s    string  // string
	a    []Value // slice elements (reference semantics)
}

// Int returns an integer value.
func Int(n int64) Value { return Value{Kind: KindInt, n: uint64(n)} }

// Float returns a floating-point value.
func Float(f float64) Value { return Value{Kind: KindFloat, n: math.Float64bits(f)} }

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
func (v Value) AsInt() int64 { return int64(v.n) }

// AsFloat returns the value as a float64.
func (v Value) AsFloat() float64 { return math.Float64frombits(v.n) }

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
		return strconv.FormatInt(int64(v.n), 10)
	case KindFloat:
		return strconv.FormatFloat(math.Float64frombits(v.n), 'g', -1, 64)
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
