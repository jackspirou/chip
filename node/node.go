package node

import "github.com/JackSpirou/chip/typ"

// A General Node Descriptor
type Node interface {
	Type() typ.Typ
	String() string
}
