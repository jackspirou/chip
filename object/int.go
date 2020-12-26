package object

import (
	"fmt"

	"github.com/jackspirou/chip/types"
)

type Integer struct {
	object
	Value int
}

// NewInteger returns a new node based on token and type
func NewInteger() *Integer {
	return &Integer{object: object{t: types.Integer{}}}
}

func (i *Integer) String() string {
	return fmt.Sprintf("%d", i.Value)
}
