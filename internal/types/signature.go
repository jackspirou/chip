package types

import "strings"

// Signature is the type of a function: its parameter types and an optional
// result type (nil means the function returns no value).
type Signature struct {
	Params []Type
	Result Type
}

func (*Signature) typ() {}

func (s *Signature) String() string {
	var b strings.Builder
	b.WriteString("func(")
	for i, p := range s.Params {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(p.String())
	}
	b.WriteString(")")
	if s.Result != nil {
		b.WriteString(" " + s.Result.String())
	}
	return b.String()
}
