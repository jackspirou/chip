package object

import (
	"github.com/jackspirou/chip/types"
)

// Object represents an Object descriptor.
type Object interface {
	Type() types.Type
	String() string
}

type object struct {
	t types.Type
}

func (o object) Type() types.Type {
	return o.t
}

func (o object) String() string {
	return o.t.String()
}
