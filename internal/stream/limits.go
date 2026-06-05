package stream

import (
	"context"
	"fmt"

	"github.com/jackspirou/chip/internal/diag"
	"github.com/jackspirou/chip/internal/token"
)

// Limits bounds a run's resource use (plan §4.1). A zero field imposes no limit
// on that dimension; a zero Depth keeps the engine's default call-depth limit.
// Caps are cooperative: the single-goroutine engine enforces them between steps
// and on output, which keeps Steps deterministic for a given program and input.
type Limits struct {
	Steps    int64 // max interpreter steps; 0 = unlimited
	OutBytes int64 // max bytes written to out; 0 = unlimited
	Depth    int   // max call depth; 0 = engine default (maxCallDepth)
}

// ResourceError is raised when a run trips a cooperative resource cap: the
// wall-clock timeout (--timeout), the step budget (--max-steps), or the output
// budget (--max-output). It carries the position of the step at which the cap
// tripped, so the fault is located like any other. A call-depth breach is *not*
// a ResourceError: it stays a RuntimeError ("call stack too deep"), a bug-first
// runtime fault, never a 125 cap (plan §4.1).
type ResourceError struct {
	Pos token.Pos
	Msg string
}

func (e ResourceError) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Pos.Line, e.Pos.Column, e.Msg)
}

// Diagnostics renders the cap breach as a single resource-phase diagnostic, so
// the renderers and the exit-code mapper can classify it (RES001/2/3) without
// type-switching. It mirrors RuntimeError.Diagnostics in the resource phase.
func (e ResourceError) Diagnostics() []diag.Diagnostic {
	return []diag.Diagnostic{{
		Severity: diag.SeverityError,
		Phase:    diag.PhaseResource,
		Message:  e.Msg,
		Primary: diag.Primary{
			IsPrimary: true,
			Line:      e.Pos.Line,
			Column:    e.Pos.Column,
			Offset:    e.Pos.Offset,
		},
	}}
}

// applyLimits configures the engine's caps from a context and a Limits before a
// run. It records the context only when it can actually fire (a non-nil Done
// channel), so the default, uncapped path pays nothing per step — no select on a
// channel that never closes. A zero Depth keeps the default set in newEngine.
func (e *engine) applyLimits(ctx context.Context, lim Limits) {
	if ctx != nil && ctx.Done() != nil {
		e.ctx = ctx
	}
	e.maxSteps = lim.Steps
	e.maxOut = lim.OutBytes
	if lim.Depth > 0 {
		e.maxDepth = lim.Depth
	}
}

// step accounts for one unit of interpreter work and enforces the wall-clock and
// step-count caps. It is called at the head of every statement and once per loop
// iteration — the points at which an otherwise-unbounded run makes progress — so
// a runaway program is stopped cooperatively between steps (D9); pos locates the
// step for the diagnostic. With no caps engaged (the default) it is a nil check
// and a compare that both fall through, so the hot path is untouched.
func (e *engine) step(pos token.Pos) error {
	if e.ctx != nil {
		select {
		case <-e.ctx.Done():
			return ResourceError{Pos: pos, Msg: "execution timed out"}
		default:
		}
	}
	if e.maxSteps > 0 {
		e.steps++
		if e.steps > e.maxSteps {
			return ResourceError{Pos: pos, Msg: fmt.Sprintf("step budget exhausted (--max-steps=%d)", e.maxSteps)}
		}
	}
	return nil
}

// emit writes a print line to the program's output, enforcing the --max-output
// cap. With no cap it is a plain Write whose error is ignored, matching print's
// fire-and-forget semantics (so the default path is byte-identical). Under a cap
// it writes only up to the remaining budget — truncating the line rather than
// dropping it — then reports RES003 so a runaway print cannot flood the
// consumer's context. pos locates the offending print.
func (e *engine) emit(pos token.Pos, buf []byte) error {
	if e.maxOut <= 0 {
		_, _ = e.out.Write(buf)
		return nil
	}
	remaining := e.maxOut - e.outBytes
	if int64(len(buf)) <= remaining {
		n, _ := e.out.Write(buf)
		e.outBytes += int64(n)
		return nil
	}
	if remaining > 0 {
		n, _ := e.out.Write(buf[:remaining])
		e.outBytes += int64(n)
	}
	return ResourceError{Pos: pos, Msg: fmt.Sprintf("output limit reached (--max-output=%d)", e.maxOut)}
}
