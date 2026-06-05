package fix_test

import (
	"strings"
	"testing"

	"github.com/jackspirou/chip/internal/analyze"
	"github.com/jackspirou/chip/internal/diag"
	"github.com/jackspirou/chip/internal/fix"
)

// unreachableProg type-checks clean but has a dead statement after the first
// return — the canonical Machine fix (removing it never changes behavior).
const unreachableProg = `package main
func main() { print(f()) }
func f() int {
    return 1
    return 2
}`

// TestApplyRemovesUnreachable applies the safe set and checks the dead tail is
// gone and the result is what check would accept.
func TestApplyRemovesUnreachable(t *testing.T) {
	out, applied, skipped, err := fix.Apply([]byte(unreachableProg), "")
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(applied) != 1 {
		t.Fatalf("applied %d fixes, want 1: %+v", len(applied), applied)
	}
	if applied[0].Applicability != diag.Machine {
		t.Errorf("applied fix applicability = %q, want machine", applied[0].Applicability)
	}
	if len(skipped) != 0 {
		t.Errorf("skipped %d fixes, want 0: %+v", len(skipped), skipped)
	}
	want := `package main
func main() { print(f()) }
func f() int {
    return 1
}`
	if string(out) != want {
		t.Errorf("after fix:\n%q\nwant:\n%q", out, want)
	}
	// The fixed source must now pass static analysis end to end.
	if rep := analyze.Source(out); !rep.OK() {
		t.Errorf("fixed source still has diagnostics: %+v", rep.Diagnostics)
	}
}

// TestApplyLeavesMaybeUnchanged checks the safety guardrail: a "did you mean"
// (Maybe) is never auto-applied, so the source is returned unchanged, but the fix
// is reported as skipped so a caller can still see it.
func TestApplyLeavesMaybeUnchanged(t *testing.T) {
	src := maybeProg(t)
	out, applied, skipped, err := fix.Apply([]byte(src), "")
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if string(out) != src {
		t.Errorf("source changed despite only a Maybe fix:\n%q", out)
	}
	if len(applied) != 0 {
		t.Errorf("applied %d fixes, want 0 (Maybe is not auto-applied)", len(applied))
	}
	if len(skipped) != 1 {
		t.Fatalf("skipped %d fixes, want 1", len(skipped))
	}
	if skipped[0].Applicability != diag.Maybe {
		t.Errorf("skipped fix applicability = %q, want maybe", skipped[0].Applicability)
	}
}

// TestBuildPlanListsUnapplied checks the dry run: it changes nothing, counts the
// skipped Maybe, and lists its edit so a harness can decide to opt in.
func TestBuildPlanListsUnapplied(t *testing.T) {
	src := maybeProg(t)
	plan := fix.BuildPlan([]byte(src), "")
	if plan.Applied != 0 {
		t.Errorf("plan.Applied = %d, want 0", plan.Applied)
	}
	if plan.Skipped != 1 {
		t.Fatalf("plan.Skipped = %d, want 1", plan.Skipped)
	}
	if len(plan.Fixes) != 1 {
		t.Fatalf("plan has %d fixes, want 1", len(plan.Fixes))
	}
	pf := plan.Fixes[0]
	if pf.Apply {
		t.Error("planned Maybe fix marked apply=true, want false")
	}
	if len(pf.Edits) == 0 {
		t.Error("planned fix lists no edits")
	}
}

// TestPartitionAllOptsClassIn checks the --all <fixId> override: a Maybe fix with
// a matching FixID joins the apply set, while a non-matching id leaves it skipped.
func TestPartitionAllOptsClassIn(t *testing.T) {
	fixes := []fix.Fix{
		{Code: "LNT004", Applicability: diag.Machine, FixID: "unreachable"},
		{Code: "NAM001", Applicability: diag.Maybe, FixID: "didyoumean"},
	}

	apply, skip := fix.Partition(fixes, "")
	if len(apply) != 1 || len(skip) != 1 {
		t.Fatalf("default: apply=%d skip=%d, want 1/1", len(apply), len(skip))
	}

	apply, skip = fix.Partition(fixes, "didyoumean")
	if len(apply) != 2 || len(skip) != 0 {
		t.Fatalf("--all didyoumean: apply=%d skip=%d, want 2/0", len(apply), len(skip))
	}

	// An empty FixID must never match an empty allFixID by accident.
	apply, _ = fix.Partition([]fix.Fix{{Applicability: diag.Maybe, FixID: ""}}, "")
	if len(apply) != 0 {
		t.Errorf("empty FixID applied with empty --all: apply=%d, want 0", len(apply))
	}
}

// maybeProg returns a program whose only fixable issue is a "did you mean"
// (Maybe): a misspelled reference to an in-scope variable. It asserts the fixture
// actually yields exactly one Maybe fix and no Machine fix, so the tests above
// rest on a verified shape rather than an assumption about the suggester.
func maybeProg(t *testing.T) string {
	t.Helper()
	const src = `package main
func main() {
    answer := 42
    print(anser)
}`
	var maybe, machine int
	for _, f := range fix.Gather([]byte(src)) {
		switch f.Applicability {
		case diag.Maybe:
			maybe++
		case diag.Machine:
			machine++
		}
	}
	if maybe != 1 || machine != 0 {
		t.Fatalf("fixture not a clean single-Maybe case: maybe=%d machine=%d", maybe, machine)
	}
	if !strings.Contains(src, "anser") {
		t.Fatal("fixture lost its typo")
	}
	return src
}
