package stream

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/jackspirou/chip/internal/diag"
)

// runLimited streams src under ctx and lim, capturing stdout and the run error.
func runLimited(t *testing.T, ctx context.Context, src string, lim Limits) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := RunWithLimits(ctx, strings.NewReader(src), &out, DirLoader("."), lim)
	return out.String(), err
}

// classify is the code a ResourceError explains as — the same path the renderers
// and the CLI exit mapper take.
func classify(t *testing.T, err error) string {
	t.Helper()
	d, ok := err.(interface {
		Diagnostics() []diag.Diagnostic
	})
	if !ok {
		t.Fatalf("error %T (%v) is not a diag.Diagnoser", err, err)
	}
	ds := d.Diagnostics()
	if len(ds) != 1 {
		t.Fatalf("got %d diagnostics, want 1", len(ds))
	}
	return diag.Classify(ds[0])
}

// A4: a runaway loop trips the step budget as RES002, and the cap is
// deterministic — the same program under the same budget stops at the same place
// with the same output every run (the property a repair loop relies on).
func TestMaxStepsCapDeterministic(t *testing.T) {
	const src = "func run() { i := 0\nfor i < 100000000 { i = i + 1 } }\nrun()\n"
	out1, err1 := runLimited(t, context.Background(), src, Limits{Steps: 100000})
	out2, err2 := runLimited(t, context.Background(), src, Limits{Steps: 100000})

	for run, err := range map[string]error{"run1": err1, "run2": err2} {
		re, ok := err.(ResourceError)
		if !ok {
			t.Fatalf("%s error = %T (%v), want ResourceError", run, err, err)
		}
		if !strings.Contains(re.Msg, "step budget exhausted") {
			t.Fatalf("%s msg = %q, want it to mention the step budget", run, re.Msg)
		}
	}
	if err1.Error() != err2.Error() {
		t.Fatalf("step cap not deterministic: %q vs %q", err1, err2)
	}
	if out1 != out2 {
		t.Fatalf("output not deterministic: %q vs %q", out1, out2)
	}
	if got := classify(t, err1); got != "RES002" {
		t.Fatalf("Classify = %q, want RES002", got)
	}
}

// An unbounded budget (the default) lets the same shape run to completion: the
// cap only bites when set.
func TestNoStepCapRunsToCompletion(t *testing.T) {
	const src = "func main() { sum := 0\ni := 0\nfor i < 1000 { sum = sum + i\ni = i + 1 }\nprint(sum) }\n"
	out, err := runLimited(t, context.Background(), src, Limits{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "499500\n" {
		t.Fatalf("output = %q, want %q", out, "499500\n")
	}
}

// The output cap truncates mid-line rather than dropping the line, stops at
// exactly the byte budget, and explains as RES003. Each iteration prints "x\n"
// (2 bytes); a 5-byte cap admits two full lines and one more byte.
func TestMaxOutputCapTruncates(t *testing.T) {
	const src = "func main() { i := 0\nfor i < 100 { print(\"x\")\ni = i + 1 } }\n"
	out, err := runLimited(t, context.Background(), src, Limits{OutBytes: 5})
	re, ok := err.(ResourceError)
	if !ok {
		t.Fatalf("error = %T (%v), want ResourceError", err, err)
	}
	if out != "x\nx\nx" {
		t.Fatalf("output = %q, want %q (truncated at the cap)", out, "x\nx\nx")
	}
	if len(out) != 5 {
		t.Fatalf("output is %d bytes, want exactly 5", len(out))
	}
	if got := classify(t, re); got != "RES003" {
		t.Fatalf("Classify = %q, want RES003", got)
	}
}

// A cancelled context stops the run between steps and is reported as a timeout
// (RES001) with no output. Pre-cancelling makes the wall-clock path deterministic
// in a test — the first step sees Done and stops, so the infinite loop never
// spins.
func TestTimeoutCapStopsRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out, err := runLimited(t, ctx, "func main() { for { } }\n", Limits{})
	re, ok := err.(ResourceError)
	if !ok {
		t.Fatalf("error = %T (%v), want ResourceError", err, err)
	}
	if !strings.Contains(re.Msg, "timed out") {
		t.Fatalf("msg = %q, want a timeout", re.Msg)
	}
	if out != "" {
		t.Fatalf("output = %q, want empty", out)
	}
	if got := classify(t, re); got != "RES001" {
		t.Fatalf("Classify = %q, want RES001", got)
	}
}

// --max-depth tunes the recursion guard but a breach stays a runtime fault
// ("call stack too deep", RUN003, exit 1), not a 125 cap: deep recursion is a
// bug to fix, not a budget to raise (plan §4.1).
func TestMaxDepthBreachIsRuntimeFault(t *testing.T) {
	const src = "func rec(n int) int { if n == 0 { return 0 }\nreturn rec(n - 1) }\nfunc main() { print(rec(50)) }\n"
	_, err := runLimited(t, context.Background(), src, Limits{Depth: 10})
	re, ok := err.(RuntimeError)
	if !ok {
		t.Fatalf("error = %T (%v), want RuntimeError (a depth breach is a runtime fault, not a cap)", err, err)
	}
	if !strings.Contains(re.Msg, "call stack too deep") {
		t.Fatalf("msg = %q, want %q", re.Msg, "call stack too deep")
	}
	if got := classify(t, re); got != "RUN003" {
		t.Fatalf("Classify = %q, want RUN003", got)
	}
}

// A depth budget above what the program needs lets the same recursion complete:
// the knob raises and lowers the same guard.
func TestMaxDepthAllowsWithinBudget(t *testing.T) {
	const src = "func rec(n int) int { if n == 0 { return 0 }\nreturn rec(n - 1) }\nfunc main() { print(rec(50)) }\n"
	out, err := runLimited(t, context.Background(), src, Limits{Depth: 1000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "0\n" {
		t.Fatalf("output = %q, want %q", out, "0\n")
	}
}
