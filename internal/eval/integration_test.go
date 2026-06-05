package eval

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackspirou/chip/internal/diag"
)

// chipBin is the path to the chip binary built once in TestMain, or "" if the
// build failed (the integration tests then skip rather than fail spuriously).
var chipBin string

func TestMain(m *testing.M) {
	os.Exit(buildAndRun(m))
}

// buildAndRun builds the chip binary to a temp dir, sets chipBin, runs the tests,
// and cleans up. It is split from TestMain so the deferred RemoveAll runs before
// os.Exit (which skips defers). A build failure is reported but not fatal: the
// in-process tests still run, and the integration tests skip.
func buildAndRun(m *testing.M) int {
	dir, err := os.MkdirTemp("", "chip-eval-bin")
	if err != nil {
		fmt.Fprintln(os.Stderr, "eval: mktemp:", err)
		return m.Run()
	}
	defer os.RemoveAll(dir)

	bin := filepath.Join(dir, "chip")
	build := exec.Command("go", "build", "-o", bin, "github.com/jackspirou/chip/cmd/chip")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "eval: build chip:", err)
		return m.Run()
	}
	chipBin = bin
	return m.Run()
}

// runChip runs the built binary with args, feeding stdin, and returns stdout,
// stderr, and the process exit code. It skips the test if the binary wasn't built.
func runChip(t *testing.T, stdin string, args ...string) (stdout, stderr string, exit int) {
	t.Helper()
	if chipBin == "" {
		t.Skip("chip binary not built")
	}
	cmd := exec.Command(chipBin, args...)
	cmd.Stdin = strings.NewReader(stdin)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			t.Fatalf("running chip %v: %v", args, err)
		}
		exit = ee.ExitCode()
	}
	return out.String(), errb.String(), exit
}

// TestIntegrationRunExitCodes drives the corpus through the real binary: every
// broken program faults static (exit 2), every fixed program runs clean (exit 0)
// and prints exactly Want. This is the end-to-end mirror of the in-process
// validator, proving the harness's notion of "solved" matches the shipped CLI.
func TestIntegrationRunExitCodes(t *testing.T) {
	for _, c := range Corpus() {
		t.Run(c.Name, func(t *testing.T) {
			if _, _, exit := runChip(t, c.Broken, "run", "-"); exit != 2 {
				t.Errorf("broken: run exit = %d, want 2 (static fault)", exit)
			}
			out, stderr, exit := runChip(t, c.Fixed, "run", "-")
			if exit != 0 {
				t.Errorf("fixed: run exit = %d, want 0; stderr=%s", exit, stderr)
			}
			if out != c.Want {
				t.Errorf("fixed: output = %q, want %q", out, c.Want)
			}
		})
	}
}

// TestIntegrationJSONResultShape checks the machine envelope: `run --format=json`
// on a broken program writes nothing to stdout, exits 2, and emits a Result on
// stderr that parses with ok=false and the expected code.
func TestIntegrationJSONResultShape(t *testing.T) {
	c := Corpus()[0] // undefined-name → NAM001
	stdout, stderr, exit := runChip(t, c.Broken, "run", "--format=json", "-")
	if exit != 2 {
		t.Errorf("exit = %d, want 2", exit)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty (program faulted before printing)", stdout)
	}
	var res diag.Result
	if err := json.Unmarshal([]byte(strings.TrimSpace(stderr)), &res); err != nil {
		t.Fatalf("stderr did not parse as diag.Result: %v\n%s", err, stderr)
	}
	if res.OK {
		t.Errorf("Result.OK = true, want false")
	}
	if len(res.Diagnostics) == 0 || res.Diagnostics[0].Code != "NAM001" {
		t.Errorf("first diagnostic code = %q, want NAM001", firstCode(res.Diagnostics))
	}
}

// TestIntegrationSaveOnErrorCaptures checks the --save-on-error capture: a broken
// run faults (exit 2) and writes the full source — every line, byte-identical — to
// the requested path, so a harness can recover exactly what it streamed.
func TestIntegrationSaveOnErrorCaptures(t *testing.T) {
	if chipBin == "" {
		t.Skip("chip binary not built")
	}
	c := Corpus()[0]
	path := filepath.Join(t.TempDir(), "captured.chp")
	_, _, exit := runChip(t, c.Broken, "run", "--save-on-error="+path, "-")
	if exit != 2 {
		t.Errorf("exit = %d, want 2", exit)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading saved file: %v", err)
	}
	if string(saved) != c.Broken {
		t.Errorf("saved source != streamed source\n got: %q\nwant: %q", string(saved), c.Broken)
	}
}

// runawayProgram loops forever without growing the stack, so it exhausts the step
// budget (RES002) rather than the call-depth limit.
const runawayProgram = `func main() {
    i := 0
    for i < 1 {
        i = 0
    }
    print(i)
}
`

// TestIntegrationStepCap checks the resource cap: a runaway under --max-steps
// terminates deterministically with exit 125 and a RES002 diagnostic, and prints
// nothing — the guarantee that lets the eval rig run model-generated code safely.
func TestIntegrationStepCap(t *testing.T) {
	stdout, stderr, exit := runChip(t, runawayProgram, "run", "--max-steps=100000", "-")
	if exit != 125 {
		t.Errorf("exit = %d, want 125 (resource cap)", exit)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "RES002") {
		t.Errorf("stderr missing RES002:\n%s", stderr)
	}
}
