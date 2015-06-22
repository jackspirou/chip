package node

import "github.com/jackspirou/chip/src/chip/types"

// A General Node Descriptor
type Node interface {
	Type() types.Typ
	String() string
}
