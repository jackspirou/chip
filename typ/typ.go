package typ

import "github.com/JackSpirou/chip/token"

// Typ Interface
type Typ interface {
	Type() token.Tokint
	String() string
}
