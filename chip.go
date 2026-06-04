// Package chip embeds the chip toy language: it parses, type-checks, compiles,
// and runs chip source from Go.
//
// The command-line tool lives in cmd/chip; this package is the library entry
// point for embedding chip in other programs. The implementation lives under
// internal/ and is deliberately not part of the public API.
package chip

import (
	"bytes"
	"fmt"
	"io"

	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/check"
	"github.com/jackspirou/chip/internal/code"
	"github.com/jackspirou/chip/internal/compiler"
	"github.com/jackspirou/chip/internal/format"
	"github.com/jackspirou/chip/internal/lint"
	"github.com/jackspirou/chip/internal/parser"
	"github.com/jackspirou/chip/internal/vm"
)

// Diagnostic is a positioned message produced while processing chip source.
type Diagnostic struct {
	Line int
	Col  int
	Msg  string
}

// String formats the diagnostic as "line:col: message".
func (d Diagnostic) String() string {
	return fmt.Sprintf("%d:%d: %s", d.Line, d.Col, d.Msg)
}

// Error reports one or more diagnostics from compiling or running chip source.
type Error struct {
	Diagnostics []Diagnostic
}

// Error implements the error interface.
func (e *Error) Error() string {
	switch len(e.Diagnostics) {
	case 0:
		return "chip: unknown error"
	case 1:
		return e.Diagnostics[0].String()
	default:
		return fmt.Sprintf("%s (and %d more)", e.Diagnostics[0], len(e.Diagnostics)-1)
	}
}

// Program is a compiled chip program, ready to run.
type Program struct {
	prog *code.Program
}

// Compile parses, type-checks, and compiles chip source into a Program. On
// failure it returns an *Error carrying positioned diagnostics.
func Compile(src []byte) (*Program, error) {
	file, err := parse(src)
	if err != nil {
		return nil, asError(err)
	}
	info, err := check.Check(file)
	if err != nil {
		return nil, asError(err)
	}
	prog, err := compiler.Compile(file, info)
	if err != nil {
		return nil, asError(err)
	}
	return &Program{prog: prog}, nil
}

// Run executes the program, writing its output to out.
func (p *Program) Run(out io.Writer) error {
	if err := vm.Run(p.prog, out); err != nil {
		return asError(err)
	}
	return nil
}

// Run compiles and executes chip source, writing program output to out.
func Run(src []byte, out io.Writer) error {
	p, err := Compile(src)
	if err != nil {
		return err
	}
	return p.Run(out)
}

// Format returns src rewritten in canonical form. Like gofmt, it only requires
// the source to parse; it does not type-check.
func Format(src []byte) ([]byte, error) {
	out, err := format.Source(src)
	if err != nil {
		return nil, asError(err)
	}
	return out, nil
}

// Lint type-checks src and returns any style and correctness issues.
func Lint(src []byte) ([]Diagnostic, error) {
	file, err := parse(src)
	if err != nil {
		return nil, asError(err)
	}
	info, err := check.Check(file)
	if err != nil {
		return nil, asError(err)
	}
	return diagnostics(lint.Lint(file, info)), nil
}

func parse(src []byte) (*ast.File, error) {
	p, err := parser.New(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	return p.Parse()
}

// asError converts a known internal positioned error into a public *Error, or
// returns err unchanged when it carries no diagnostics.
func asError(err error) error {
	if err == nil {
		return nil
	}
	if ds := diagnostics(err); len(ds) > 0 {
		return &Error{Diagnostics: ds}
	}
	return err
}

// diagnostics extracts positioned diagnostics from any known internal error or
// issue list.
func diagnostics(v any) []Diagnostic {
	var ds []Diagnostic
	switch e := v.(type) {
	case parser.ErrorList:
		for _, d := range e {
			ds = append(ds, Diagnostic{Line: d.Pos.Line, Col: d.Pos.Column, Msg: d.Msg})
		}
	case check.ErrorList:
		for _, d := range e {
			ds = append(ds, Diagnostic{Line: d.Pos.Line, Col: d.Pos.Column, Msg: d.Msg})
		}
	case lint.IssueList:
		for _, d := range e {
			ds = append(ds, Diagnostic{Line: d.Pos.Line, Col: d.Pos.Column, Msg: d.Msg})
		}
	case vm.RuntimeError:
		ds = append(ds, Diagnostic{Line: e.Pos.Line, Col: e.Pos.Column, Msg: e.Msg})
	}
	return ds
}
