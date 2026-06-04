package check

import (
	"fmt"

	"github.com/jackspirou/chip/internal/token"
)

// Error is a type-checking error at a source position.
type Error struct {
	Pos token.Pos
	Msg string
}

// Error implements the error interface.
func (e Error) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Pos.Line, e.Pos.Column, e.Msg)
}

// ErrorList is a collection of type-checking errors.
type ErrorList []Error

// Error implements the error interface.
func (l ErrorList) Error() string {
	switch len(l) {
	case 0:
		return "no errors"
	case 1:
		return l[0].Error()
	default:
		return fmt.Sprintf("%s (and %d more errors)", l[0], len(l)-1)
	}
}

// Err returns an error equivalent to the list, or nil if the list is empty.
func (l ErrorList) Err() error {
	if len(l) == 0 {
		return nil
	}
	return l
}
