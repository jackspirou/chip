// Package fix turns the repair suggestions a diagnostic carries into applied
// source edits — the engine behind `chip fix`. It gathers the fixes the static
// analysis and linter propose, decides which are safe to apply automatically,
// and splices them in.
//
// The safety boundary is applicability (D6): only Machine fixes — edits that are
// unconditionally behavior-preserving, like removing provably-dead code — are
// applied by default. Maybe fixes (a "did you mean" guess) and Placeholder fixes
// (a value the tool cannot know) are never applied blind; they are reported in
// the plan so a caller can opt a whole class in with --all <fixId>. This mirrors
// cargo fix / gopls -fix: apply the certain, surface the rest.
package fix

import (
	"bytes"

	"github.com/jackspirou/chip/internal/analyze"
	"github.com/jackspirou/chip/internal/check"
	"github.com/jackspirou/chip/internal/diag"
	"github.com/jackspirou/chip/internal/lint"
	"github.com/jackspirou/chip/internal/parser"
)

// Fix is one applicable repair extracted from a diagnostic: the concrete byte
// edits plus the routing metadata `chip fix` needs to decide whether to apply it.
type Fix struct {
	Code          string             // diagnostic code (e.g. NAM001, LNT004)
	Message       string             // the suggestion's message ("remove unreachable code")
	Applicability diag.Applicability // machine / maybe / placeholder
	FixID         string             // repair class for --all (empty if the fix carries no Repair)
	Line          int                // 1-based line the diagnostic points at
	Column        int                // 1-based column
	Edits         []diag.Edit        // the byte-range splices that realize the fix
}

// Gather collects every applicable fix in src, from both the static analysis
// (parse/type errors with "did you mean" suggestions) and the linter (unused
// names, dead code). Only suggestions that carry concrete edits become fixes; a
// purely advisory message is skipped. A program that does not parse yields only
// whatever its parse-error diagnostics suggest (usually nothing), so Gather never
// fails — it returns the fixes it can extract.
func Gather(src []byte) []Fix {
	var fixes []Fix
	collect := func(ds []diag.Diagnostic) {
		for _, d := range diag.WithCodes(ds) {
			for _, s := range d.Suggestions {
				if len(s.Edit) == 0 {
					continue // advisory only — nothing to apply
				}
				f := Fix{
					Code:          d.Code,
					Message:       s.Message,
					Applicability: s.Applicability,
					Line:          d.Primary.Line,
					Column:        d.Primary.Column,
					Edits:         s.Edit,
				}
				if d.Repair != nil {
					f.FixID = d.Repair.FixID
				}
				fixes = append(fixes, f)
			}
		}
	}
	// Static analysis carries the "did you mean" suggestions (Maybe). It runs
	// over a recovered tree even when the program is broken, so it always reports.
	collect(analyze.Source(src).Diagnostics)
	// Lint carries the Machine fixes (dead-code removal). It only runs on a
	// program that parses and type-checks; a broken one has analysis errors to
	// fix first, and those are already in hand above.
	collect(lintFixes(src))
	return fixes
}

// lintFixes parses, type-checks, and lints src, returning the lint findings as
// diagnostics — or nil if the program does not parse or type-check (lint cannot
// run, and analysis already reported those errors).
func lintFixes(src []byte) []diag.Diagnostic {
	p, err := parser.New(bytes.NewReader(src))
	if err != nil {
		return nil
	}
	f, err := p.Parse()
	if err != nil {
		return nil
	}
	info, err := check.Check(f)
	if err != nil {
		return nil
	}
	return lint.Lint(f, info).Diagnostics()
}

// Partition splits fixes into those to apply now and those to leave for the
// caller. A Machine fix always applies. A fix whose FixID matches allFixID is
// applied too, regardless of applicability — the --all <fixId> override that lets
// a caller opt a whole repair class in (e.g. accept every "did you mean").
func Partition(fixes []Fix, allFixID string) (apply, skip []Fix) {
	for _, f := range fixes {
		if f.Applicability == diag.Machine || (allFixID != "" && f.FixID != "" && f.FixID == allFixID) {
			apply = append(apply, f)
		} else {
			skip = append(skip, f)
		}
	}
	return apply, skip
}

// Apply gathers the fixes in src, applies the safe set (Machine, plus the --all
// class), and returns the rewritten source together with the fixes it applied and
// the ones it left. The edits are spliced in one pass; overlapping edits are
// refused by the applier rather than guessed, surfacing as an error.
func Apply(src []byte, allFixID string) (out []byte, applied, skipped []Fix, err error) {
	apply, skip := Partition(Gather(src), allFixID)
	var edits []diag.Edit
	for _, f := range apply {
		edits = append(edits, f.Edits...)
	}
	out, err = diag.ApplyEdits(src, edits)
	if err != nil {
		return nil, nil, nil, err
	}
	return out, apply, skip, nil
}

// PlannedFix is one fix as it appears in a --plan: where it is, what it does, its
// applicability, and whether `chip fix` would apply it automatically.
type PlannedFix struct {
	Code          string      `json:"code,omitempty"`
	Message       string      `json:"message"`
	Applicability string      `json:"applicability"`
	FixID         string      `json:"fixId,omitempty"`
	Line          int         `json:"line"`
	Column        int         `json:"column"`
	Edits         []diag.Edit `json:"edits"`
	Apply         bool        `json:"apply"` // would chip fix apply this automatically?
}

// Plan is the dry-run shape (--plan --json): every fix found, each tagged with
// whether it would be auto-applied, plus the apply/skip counts. A harness reads
// it to see the unapplied edits and decide which Maybe class to opt into (via
// --all <fixId>) before re-running.
type Plan struct {
	Fixes   []PlannedFix `json:"fixes"`
	Applied int          `json:"applied"`
	Skipped int          `json:"skipped"`
}

// BuildPlan computes the dry run for src under the given --all class: every fix,
// tagged with whether it would be applied, without changing the source.
func BuildPlan(src []byte, allFixID string) Plan {
	apply, skip := Partition(Gather(src), allFixID)
	plan := Plan{
		Fixes:   make([]PlannedFix, 0, len(apply)+len(skip)),
		Applied: len(apply),
		Skipped: len(skip),
	}
	for _, f := range apply {
		plan.Fixes = append(plan.Fixes, planned(f, true))
	}
	for _, f := range skip {
		plan.Fixes = append(plan.Fixes, planned(f, false))
	}
	return plan
}

func planned(f Fix, apply bool) PlannedFix {
	app := string(f.Applicability)
	if app == "" {
		app = string(diag.Unspecified)
	}
	return PlannedFix{
		Code:          f.Code,
		Message:       f.Message,
		Applicability: app,
		FixID:         f.FixID,
		Line:          f.Line,
		Column:        f.Column,
		Edits:         f.Edits,
		Apply:         apply,
	}
}
