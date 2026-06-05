package parser

import (
	"fmt"

	"github.com/jackspirou/chip/internal/diag"
	"github.com/jackspirou/chip/internal/token"
)

// Error is a parse error at a source position.
type Error struct {
	Pos token.Pos
	Msg string
}

// Error implements the error interface.
func (e Error) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Pos.Line, e.Pos.Column, e.Msg)
}

// ErrorList is a collection of parse errors.
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

// Diagnostics renders the list as structured diagnostics: every parse error is
// an error in the parse phase. It implements diag.Diagnoser so the renderers can
// present parse errors without type-switching.
func (l ErrorList) Diagnostics() []diag.Diagnostic {
	ds := make([]diag.Diagnostic, len(l))
	for i, e := range l {
		ds[i] = diag.Diagnostic{
			Severity: diag.SeverityError,
			Phase:    diag.PhaseParse,
			Message:  e.Msg,
			Primary: diag.Primary{
				IsPrimary: true,
				Line:      e.Pos.Line,
				Column:    e.Pos.Column,
				Offset:    e.Pos.Offset,
			},
		}
	}
	return ds
}
