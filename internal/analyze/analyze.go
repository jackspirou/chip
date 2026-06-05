// Package analyze runs chip's batch static analysis — parse plus local type
// checking — and returns cascade-pruned structured diagnostics. It is the shared
// core behind the public chip.Check (which down-converts to the library's flat
// Diagnostic) and the CLI's `chip check` (which renders the rich diagnostics with
// codes and "did you mean" suggestions), so both get one pruning pass and one set
// of positions, messages, and hints.
//
// Boundary (D3): imports are opaque (no Loader on the batch path, so a qualified
// pkg.Member call is neither resolved nor rejected) and there is no runtime — so
// analysis reports only the static errors it can know whole-program, never a
// runtime fault. This matches the batch checker and mirrors what running would
// report before the first fault, without executing.
package analyze

import (
	"bytes"
	"sort"

	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/check"
	"github.com/jackspirou/chip/internal/diag"
	"github.com/jackspirou/chip/internal/parser"
	"github.com/jackspirou/chip/internal/token"
)

// Report is the result of Source: every static error that survived cascade
// pruning, in source order, plus the number of cascade-derived errors that were
// suppressed (so the pruning stays auditable). Diagnostics is empty when src is
// valid.
type Report struct {
	Diagnostics        []diag.Diagnostic
	SuppressedCascades int
}

// OK reports whether src passed analysis with no surviving diagnostics.
func (r Report) OK() bool { return len(r.Diagnostics) == 0 }

// Source statically analyzes src without executing it, returning every
// independent parse and type error at once (collect-all), with cascade-derived
// follow-on errors pruned: one root error per top-level construct, and the
// phantom type errors a broken construct provokes suppressed (counted in
// SuppressedCascades). Diagnostics carry their phase, byte offsets, and any
// grounded fix suggestions the checker attached.
func Source(src []byte) Report {
	file, parseErrs := parseAll(src)
	var typeErrs check.ErrorList
	if file != nil {
		if _, cerr := check.Check(file); cerr != nil {
			if cl, ok := cerr.(check.ErrorList); ok {
				typeErrs = cl
			}
		}
	}
	return prune(file, parseErrs, typeErrs)
}

// parseAll parses src and returns the (possibly partial) file together with the
// full list of parse errors, so the checker can run over the recovered tree and
// the pruner can attribute cascades to the constructs they came from.
func parseAll(src []byte) (*ast.File, parser.ErrorList) {
	p, err := parser.New(bytes.NewReader(src))
	if err != nil {
		return nil, parser.ErrorList{{Pos: token.Pos{Line: 1, Column: 1}, Msg: err.Error()}}
	}
	file, perr := p.Parse()
	if pl, ok := perr.(parser.ErrorList); ok {
		return file, pl
	}
	return file, nil
}

// span is a half-open byte range [lo, hi) covering one top-level construct.
type span struct{ lo, hi int }

func (s span) contains(offset int) bool { return s.lo <= offset && offset < s.hi }

// constructs returns the byte spans of every top-level construct (declarations
// and statements), sorted by start offset, so an error's offset can be attributed
// to the construct it falls in.
func constructs(file *ast.File) []span {
	if file == nil {
		return nil
	}
	cs := make([]span, 0, len(file.Decls)+len(file.Stmts))
	for _, d := range file.Decls {
		cs = append(cs, span{d.Pos().Offset, d.End().Offset})
	}
	for _, s := range file.Stmts {
		cs = append(cs, span{s.Pos().Offset, s.End().Offset})
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].lo < cs[j].lo })
	return cs
}

// constructAt returns the index of the first construct (by start offset) whose
// span contains offset, or -1 if none does. Picking the first means a recovery
// artifact that bleeds into the next construct's start is attributed back to the
// broken construct it came from, not the (innocent) one it landed on.
func constructAt(cs []span, offset int) int {
	for i, c := range cs {
		if c.contains(offset) {
			return i
		}
	}
	return -1
}

// prune turns the raw parse and type errors into a cascade-pruned report of rich
// diagnostics.
//
// One parse error per top-level construct is the root cause; the rest of that
// construct's parse errors are recovery noise. A type error inside a construct
// whose syntax failed to parse is a phantom of the bad tree, so it is suppressed
// until the syntax is fixed; type errors in cleanly-parsed constructs survive, so
// genuinely independent errors are all reported. Suppressed errors are counted,
// never silently dropped.
func prune(file *ast.File, parseErrs parser.ErrorList, typeErrs check.ErrorList) Report {
	cs := constructs(file)
	var r Report

	rooted := map[int]bool{} // group key -> a parse error was already kept for it
	broken := map[int]bool{} // construct index -> its syntax is broken (has a parse error)
	for _, e := range parseErrs {
		ci := constructAt(cs, e.Pos.Offset)
		// Group by construct; errors outside every construct group by exact offset,
		// so only same-position storms collapse and distinct loose errors all
		// survive.
		key := ci
		if ci < 0 {
			key = -1 - e.Pos.Offset
		}
		if rooted[key] {
			r.SuppressedCascades++
			continue
		}
		rooted[key] = true
		if ci >= 0 {
			broken[ci] = true
		}
		r.Diagnostics = append(r.Diagnostics, parseDiag(e))
	}

	for _, e := range typeErrs {
		if ci := constructAt(cs, e.Pos.Offset); ci >= 0 && broken[ci] {
			r.SuppressedCascades++
			continue
		}
		r.Diagnostics = append(r.Diagnostics, typeDiag(e))
	}

	sortDiagnostics(r.Diagnostics)
	return r
}

// parseDiag builds a parse-phase diagnostic from a parser error.
func parseDiag(e parser.Error) diag.Diagnostic {
	return diag.Diagnostic{
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

// typeDiag builds a type-phase diagnostic from a checker error, carrying through
// any grounded fix suggestions the checker attached (for example "did you mean").
func typeDiag(e check.Error) diag.Diagnostic {
	return diag.Diagnostic{
		Severity: diag.SeverityError,
		Phase:    diag.PhaseType,
		Message:  e.Msg,
		Primary: diag.Primary{
			IsPrimary: true,
			Line:      e.Pos.Line,
			Column:    e.Pos.Column,
			Offset:    e.Pos.Offset,
		},
		Suggestions: e.Suggestions,
	}
}

// sortDiagnostics orders diagnostics by source position so output is
// deterministic and reads top-to-bottom.
func sortDiagnostics(ds []diag.Diagnostic) {
	sort.SliceStable(ds, func(i, j int) bool {
		a, b := ds[i].Primary, ds[j].Primary
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Column < b.Column
	})
}
