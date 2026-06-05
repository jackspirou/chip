package eval

import "testing"

// This file mechanically enforces the checker invariant the soundness audit was
// about: chip check is never more lenient than chip run. If chip run rejects a
// program before executing it (a static fault, exit 2), chip check must reject it
// too. check is allowed to be stricter — to reject something run would have run —
// but never the reverse, because tooling, editors, and agents lean on check to
// predict what run will do, and a check that quietly passes a program run refuses
// is lying to them.
//
// The rules that decide static rejection (operators, termination, the main
// signature) live in internal/typerules and are called from both checkers, so the
// invariant holds by construction. This test is the end-to-end proof of that
// construction through the shipped binary: it drives both subcommands over a
// corpus spanning every static-rejection class — including the four leniency gaps
// the audit closed — plus the runtime-fault boundary and a set of valid programs.

// runKind classifies what chip run does with a program, which fixes the exit codes
// both subcommands must produce.
type runKind int

const (
	// staticReject: chip run refuses the program before executing it (exit 2).
	// The invariant requires chip check to exit 2 as well.
	staticReject runKind = iota
	// runtimeFault: chip run executes the program, which then faults at runtime
	// (exit 1). No static checker can catch these, so chip check exits 0 — this is
	// chip's D3 boundary, not a leniency: run did not statically reject, so there
	// is nothing for check to have caught.
	runtimeFault
	// valid: the program runs clean (exit 0). check must also pass it (exit 0):
	// the invariant permits check to be stricter, but these are vetted-valid, so a
	// check rejection here would be a spurious over-rejection worth failing on.
	valid
)

// conformanceCase is one program with the run behavior it must exhibit. None
// import a package: a qualified call through an import is a known, deliberate
// divergence (the batch checker treats imports opaquely while the streaming
// checker resolves them), so the import path is out of scope for this invariant.
type conformanceCase struct {
	name string
	kind runKind
	src  string
}

// conformanceCorpus is the parity corpus. The staticReject block leads with the
// four gaps the audit closed (bitwise/shift, function-as-value, main-with-args,
// missing-return), then covers the static errors both checkers already caught, so
// the test guards both the fixes and the rules that were always shared.
func conformanceCorpus() []conformanceCase {
	return []conformanceCase{
		// --- the four closed leniency gaps ---
		{"bitwise-and", staticReject, `package main
func main() {
    x := 1 & 2
    print(x)
}`},
		{"shift-left", staticReject, `package main
func main() {
    x := 1 << 2
    print(x)
}`},
		{"function-as-value", staticReject, `package main
func f() int { return 1 }
func main() { print(f) }`},
		{"main-with-args", staticReject, `package main
func main(x int) { print(x) }`},
		{"missing-return", staticReject, `package main
func f() int { print(1) }
func main() { print(f()) }`},

		// --- static errors both checkers already rejected (regression guard) ---
		{"undefined-name", staticReject, `package main
func main() { print(a) }`},
		{"return-type-mismatch", staticReject, `package main
func f() int { return "x" }
func main() { print(f()) }`},
		{"wrong-arg-count", staticReject, `package main
func f(x int) {}
func main() { f(1, 2) }`},
		{"arg-type-mismatch", staticReject, `package main
func f(x int) {}
func main() { f("s") }`},
		{"non-bool-condition", staticReject, `package main
func main() { if 1 { print(1) } }`},
		{"redeclared", staticReject, `package main
func main() {
    x := 1
    x := 2
    print(x)
}`},
		{"bad-operand", staticReject, `package main
func main() {
    x := 1 + "y"
    print(x)
}`},
		{"missing-return-value", staticReject, `package main
func f() int { return }
func main() { print(f()) }`},

		// --- runtime faults: the D3 boundary (run faults, check is silent) ---
		{"division-by-zero", runtimeFault, `package main
func main() { print(1 / 0) }`},
		{"index-out-of-range", runtimeFault, `package main
func main() {
    xs := []int{1}
    print(xs[5])
}`},

		// --- valid programs: both must pass ---
		{"arithmetic", valid, `package main
func main() { print(1 + 10) }`},
		{"recursion-gcd", valid, `package main
func gcd(a int, b int) int {
    if b == 0 {
        return a
    }
    return gcd(b, a % b)
}
func main() { print(gcd(252, 105)) }`},
		{"if-else-returns", valid, `package main
func pick(x int) int {
    if x > 0 {
        return 1
    }
    return 0
}
func main() { print(pick(1)) }`},
		{"slice-loop", valid, `package main
func main() {
    xs := []int{1, 2, 3}
    i := 0
    total := 0
    for i < len(xs) {
        total = total + xs[i]
        i = i + 1
    }
    print(total)
}`},
	}
}

// TestRunCheckConformance drives every corpus program through both chip run and
// chip check on the built binary and asserts the exit codes the program's run
// behavior demands. The crux is the staticReject block: each such program faults
// static under run (exit 2), and the invariant requires check to fault too. The
// loop also asserts the invariant directly — never (run rejected, check passed) —
// so a misclassified or newly-added case still can't let a leniency through.
func TestRunCheckConformance(t *testing.T) {
	for _, c := range conformanceCorpus() {
		t.Run(c.name, func(t *testing.T) {
			_, runErr, runExit := runChip(t, c.src, "run", "-")
			_, _, checkExit := runChip(t, c.src, "check", "-")

			// The invariant, stated directly and independent of classification: if
			// run statically rejected the program, check must have too.
			if runExit == 2 && checkExit != 2 {
				t.Fatalf("chip check is more lenient than chip run: run exit=2 (rejected) but check exit=%d\nstderr from run:\n%s", checkExit, runErr)
			}

			switch c.kind {
			case staticReject:
				if runExit != 2 {
					t.Errorf("run exit = %d, want 2 (static rejection); stderr:\n%s", runExit, runErr)
				}
				if checkExit != 2 {
					t.Errorf("check exit = %d, want 2 (must match run's static rejection)", checkExit)
				}
			case runtimeFault:
				if runExit != 1 {
					t.Errorf("run exit = %d, want 1 (ran, then faulted at runtime); stderr:\n%s", runExit, runErr)
				}
				// check cannot catch a dynamic fault and is correctly silent. The
				// invariant permits a stricter check (a future constant-folder could
				// reject 1/0 at exit 2); should that land, reclassify this case.
				if checkExit != 0 {
					t.Errorf("check exit = %d, want 0 (dynamic fault is invisible to static analysis)", checkExit)
				}
			case valid:
				if runExit != 0 {
					t.Errorf("run exit = %d, want 0 (valid program); stderr:\n%s", runExit, runErr)
				}
				if checkExit != 0 {
					t.Errorf("check exit = %d, want 0 (a vetted-valid program must not be over-rejected)", checkExit)
				}
			}
		})
	}
}
