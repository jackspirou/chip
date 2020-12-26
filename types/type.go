package types

import (
	"github.com/jackspirou/chip/token"
)

// Type is the type interface
type Type interface {
	Token() token.Type
	String() string
}
