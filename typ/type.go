// Package typ describes built-in types.
package typ

import "github.com/jackspirou/chip/token"

// Type represents a type.
type Type interface {
	Token() token.Type
	Value() Type
	String() string
}
