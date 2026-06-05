package stream

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// runSealed streams main through the engine with the given loader, optionally
// sealing the prelude, and returns stdout and the run error. It is the seam an
// embedder uses to run untrusted code hermetically (plan §4.2).
func runSealed(t *testing.T, main string, loader Loader, noPrelude bool) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := RunSealed(context.Background(), strings.NewReader(main), &out, loader, Limits{}, noPrelude)
	return out.String(), err
}

// The embed posture for sealing the host (plan §4.2): a program run with a
// DenyLoader cannot import anything from the host — the loader is chip's only
// host-access vector — and the refusal names the offending path and reads
// cleanly. This is what the CLI's --no-imports installs.
func TestDenyLoaderSealsHostImports(t *testing.T) {
	_, err := runSealed(t, "import \"greet\"\nprint(1)\n", DenyLoader(), false)
	if err == nil {
		t.Fatal("import under a DenyLoader succeeded, want a refusal")
	}
	msg := err.Error()
	if !strings.Contains(msg, "imports are disabled") {
		t.Fatalf("error = %q, want it to say imports are disabled", msg)
	}
	if !strings.Contains(msg, "greet") {
		t.Fatalf("error = %q, want it to name the offending import path", msg)
	}
}

// An empty in-memory loader blocks host access just as well: no package resolves,
// so no host code loads. This is the "embed test with an empty Loader" the plan's
// §4.2 Verify calls for — host access sealed off by construction.
func TestEmptyLoaderBlocksHostAccess(t *testing.T) {
	_, err := runSealed(t, "import \"anything\"\nprint(1)\n", MapLoader(nil), false)
	if err == nil {
		t.Fatal("import under an empty loader succeeded, want a failure")
	}
	if !strings.Contains(err.Error(), "anything") {
		t.Fatalf("error = %q, want it to name the unresolved import", err.Error())
	}
}

// Sealing the host does not seal the bundled standard library: import "math" is
// resolved by the engine ahead of any loader, so math.Gcd still works under a
// DenyLoader. Bundled stdlib is pure chip code, not host access.
func TestSealedRunStdlibStillResolves(t *testing.T) {
	out, err := runSealed(t, "import \"math\"\nprint(math.Gcd(12, 18))\n", DenyLoader(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "6\n" {
		t.Fatalf("stdout = %q, want %q", out, "6\n")
	}
}

// Sealing the prelude drops the unqualified stdlib functions: a program that
// calls a prelude function (abs) without it is a clean undefined-name error,
// while the same program with the prelude loaded works. The seal is independent
// of the loader (the prelude is fed directly, not imported), so a DenyLoader run
// still has abs until --no-prelude removes it.
func TestNoPreludeSealsPrelude(t *testing.T) {
	const src = "print(abs(0 - 7))\n"

	out, err := runSealed(t, src, DenyLoader(), false)
	if err != nil {
		t.Fatalf("with prelude: unexpected error: %v", err)
	}
	if out != "7\n" {
		t.Fatalf("with prelude: stdout = %q, want %q", out, "7\n")
	}

	if _, err := runSealed(t, src, DenyLoader(), true); err == nil {
		t.Fatal("abs resolved under --no-prelude, want undefined")
	} else if !strings.Contains(err.Error(), "undefined: abs") {
		t.Fatalf("error = %q, want %q", err.Error(), "undefined: abs")
	}
}

// Sealing the prelude leaves the builtins (print, len) — they are built into the
// engine, not the prelude — so a fully hermetic run still has output and length.
func TestNoPreludeKeepsBuiltins(t *testing.T) {
	out, err := runSealed(t, "print(len(\"hello\"))\n", DenyLoader(), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "5\n" {
		t.Fatalf("stdout = %q, want %q", out, "5\n")
	}
}

// RunSealed with noPrelude=false is exactly RunWithLimits: same prelude, same
// output. This pins that adding the seal did not change the default run path
// (A7) — skipping loadStd is the only difference a seal makes.
func TestRunSealedDefaultMatchesRunWithLimits(t *testing.T) {
	const src = "print(max(3, 9))\n" // max is a prelude function

	var sealedOut, limitsOut bytes.Buffer
	if err := RunSealed(context.Background(), strings.NewReader(src), &sealedOut, DenyLoader(), Limits{}, false); err != nil {
		t.Fatalf("RunSealed: %v", err)
	}
	if err := RunWithLimits(context.Background(), strings.NewReader(src), &limitsOut, DenyLoader(), Limits{}); err != nil {
		t.Fatalf("RunWithLimits: %v", err)
	}
	if sealedOut.String() != limitsOut.String() {
		t.Fatalf("RunSealed=%q != RunWithLimits=%q", sealedOut.String(), limitsOut.String())
	}
	if sealedOut.String() != "9\n" {
		t.Fatalf("stdout = %q, want %q", sealedOut.String(), "9\n")
	}
}
