package chip

import "github.com/jackspirou/chip/internal/analyze"

// CheckReport is the result of Check: every static error that survived cascade
// pruning, in source order, plus the number of cascade-derived errors that were
// suppressed (so the pruning stays auditable). It is empty when src is valid.
type CheckReport struct {
	Diagnostics        []Diagnostic
	SuppressedCascades int
}

// OK reports whether src passed the check with no surviving diagnostics.
func (r *CheckReport) OK() bool { return len(r.Diagnostics) == 0 }

// Check statically analyzes src without executing it, returning every
// independent parse and type error at once (collect-all), with cascade-derived
// follow-on errors pruned: one root error per top-level construct, and the
// phantom type errors that a broken construct provokes are suppressed (count in
// SuppressedCascades).
//
// Check is the analysis half of run: it reports what running would report before
// the first fault, never executing. Its boundary matches the batch checker (D3):
// imports are opaque (no Loader, so a qualified pkg.Member call is not resolved or
// rejected) and there is no runtime — so it never reports a runtime fault, only
// static errors it can know whole-program.
//
// Check is the library's flat view: it down-converts the shared analysis (which
// carries phases, byte offsets, and fix suggestions) to the public
// position-and-message Diagnostic. The CLI's `chip check` consumes the rich form
// directly for codes and "did you mean".
func Check(src []byte) *CheckReport {
	rep := analyze.Source(src)
	out := &CheckReport{SuppressedCascades: rep.SuppressedCascades}
	for _, d := range rep.Diagnostics {
		out.Diagnostics = append(out.Diagnostics, Diagnostic{
			Line: d.Primary.Line,
			Col:  d.Primary.Column,
			Msg:  d.Message,
		})
	}
	return out
}
