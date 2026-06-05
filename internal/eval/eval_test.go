package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jackspirou/chip/internal/diag"
)

// TestCorpusBrokenFailsFixedSolves locks the corpus against rot: every Broken must
// fault (and yield feedback to send back), and every Fixed must run clean and print
// Want. If a language change ever makes a "broken" case valid or a "fixed" case
// fault, this fails loudly instead of the eval rig silently measuring nonsense.
func TestCorpusBrokenFailsFixedSolves(t *testing.T) {
	ctx := context.Background()
	for _, c := range Corpus() {
		t.Run(c.Name, func(t *testing.T) {
			solved, ds, got := validate(ctx, c.Broken, c.Want, FeedbackCollectAll)
			if solved {
				t.Fatalf("broken program unexpectedly solved (output %q)", got)
			}
			if len(ds) == 0 {
				t.Fatalf("broken program produced no diagnostics to feed back")
			}
			solved, _, got = validate(ctx, c.Fixed, c.Want, FeedbackCollectAll)
			if !solved {
				t.Fatalf("fixed program not solved; output %q, want %q", got, c.Want)
			}
		})
	}
}

// TestSolveCountsTurnsAndTokens checks the core measurement: a model that stays
// broken for one turn and fixes on the second takes two turns and is charged for
// both calls.
func TestSolveCountsTurnsAndTokens(t *testing.T) {
	c := Corpus()[0] // undefined-name
	m := NewScriptedModel([]string{c.Broken, c.Fixed}, 100)
	out := Solve(context.Background(), m, c, Config{Profile: ProfileAgent, MaxTurns: 3})
	if !out.Solved {
		t.Fatalf("expected solved")
	}
	if out.Turns != 2 {
		t.Errorf("turns = %d, want 2", out.Turns)
	}
	if out.Tokens != 200 {
		t.Errorf("tokens = %d, want 200 (2 calls * 100)", out.Tokens)
	}
}

// TestSolveGivesUpAtMaxTurns checks the cap: a model that never fixes the program
// stops after MaxTurns attempts, unsolved, charged for each.
func TestSolveGivesUpAtMaxTurns(t *testing.T) {
	c := Corpus()[0]
	m := NewScriptedModel([]string{c.Broken}, 50) // only ever returns the broken source
	out := Solve(context.Background(), m, c, Config{Profile: ProfileAgent, MaxTurns: 3})
	if out.Solved {
		t.Fatalf("expected unsolved")
	}
	if out.Turns != 3 {
		t.Errorf("turns = %d, want 3 (the cap)", out.Turns)
	}
	if out.Tokens != 150 {
		t.Errorf("tokens = %d, want 150 (3 calls * 50)", out.Tokens)
	}
}

// TestEvalTablePerProfile is the slice's Verify: the rig runs the whole corpus
// under each profile and emits a turns-/tokens-to-green table. With a model that
// fixes in one turn, every case goes green at one turn per profile. The table is
// logged so a human (or CI log) can read the numbers.
func TestEvalTablePerProfile(t *testing.T) {
	ctx := context.Background()
	cases := Corpus()
	var rows []Summary
	for _, p := range AllProfiles {
		outcomes := RunCorpus(ctx, func(c Case) Model {
			return NewScriptedModel([]string{c.Fixed}, 10)
		}, cases, Config{Profile: p, MaxTurns: 3})
		s := Summarize(p, "scripted", outcomes)
		if s.Solved != s.Cases {
			t.Errorf("profile %s: solved %d/%d, want all", p, s.Solved, s.Cases)
		}
		if s.AvgTurns != 1 {
			t.Errorf("profile %s: AvgTurns = %v, want 1", p, s.AvgTurns)
		}
		rows = append(rows, s)
	}
	var buf bytes.Buffer
	WriteTable(&buf, rows)
	t.Logf("turns-/tokens-to-green per profile:\n%s", buf.String())
}

// TestRenderProfiles checks the three profiles render the same diagnostic in their
// distinct shapes: agent leads with the bracketed code, human shows the caret with
// no code, and json parses back as the Result envelope.
func TestRenderProfiles(t *testing.T) {
	c := Corpus()[0] // undefined-name → NAM001 at 3:11
	_, ds, _ := validate(context.Background(), c.Broken, c.Want, FeedbackCollectAll)

	agent := render(ProfileAgent, c.Broken, ds)
	if !strings.Contains(agent, "error[NAM001]") {
		t.Errorf("agent render missing coded error line:\n%s", agent)
	}
	if !strings.Contains(agent, "found 1 error") {
		t.Errorf("agent render missing count summary:\n%s", agent)
	}

	human := render(ProfileHuman, c.Broken, ds)
	if !strings.Contains(human, "^") || !strings.Contains(human, ":3:11:") {
		t.Errorf("human render missing caret/position:\n%s", human)
	}
	if strings.Contains(human, "NAM001") {
		t.Errorf("human render should carry no code:\n%s", human)
	}

	jsonOut := render(ProfileJSON, c.Broken, ds)
	var res diag.Result
	if err := json.Unmarshal([]byte(jsonOut), &res); err != nil {
		t.Fatalf("json render did not parse as diag.Result: %v\n%s", err, jsonOut)
	}
	if res.OK {
		t.Errorf("json render: OK = true, want false for a faulting program")
	}
	if len(res.Diagnostics) == 0 || res.Diagnostics[0].Code != "NAM001" {
		t.Errorf("json render: first diagnostic code = %q, want NAM001", firstCode(res.Diagnostics))
	}
}

// TestRenderLLMProfile checks the per-model profile wiring (§5.5): both the bare
// "llm" profile and an "llm:<model>" tuning render the grounded LLM view — a strict
// superset of the agent render plus the catalog's repair hint — so the rig feeds a
// model the same grounded feedback the CLI's llm:* format emits.
func TestRenderLLMProfile(t *testing.T) {
	c := Corpus()[0] // undefined-name → NAM001
	_, ds, _ := validate(context.Background(), c.Broken, c.Want, FeedbackCollectAll)

	agent := render(ProfileAgent, c.Broken, ds)
	entry, ok := diag.Explain("NAM001")
	if !ok {
		t.Fatal("catalog has no NAM001 entry")
	}
	for _, p := range []Profile{ProfileLLM, "llm:claude-opus-4-8"} {
		llm := render(p, c.Broken, ds)
		if !strings.HasPrefix(llm, agent) {
			t.Errorf("profile %s: not a superset of agent:\nagent:\n%s\nllm:\n%s", p, agent, llm)
		}
		if !strings.Contains(llm, "how to fix:") {
			t.Errorf("profile %s: missing grounded appendix:\n%s", p, llm)
		}
		if !strings.Contains(llm, entry.Fix) {
			t.Errorf("profile %s: missing NAM001 fix guidance %q:\n%s", p, entry.Fix, llm)
		}
	}
}

// firstCode is a small helper for the failure message above.
func firstCode(ds []diag.Diagnostic) string {
	if len(ds) == 0 {
		return "(none)"
	}
	return ds[0].Code
}

// TestExtractSourceUnwrapsFences checks the reply parser: a fenced block is
// unwrapped, plain source survives byte-for-byte.
func TestExtractSourceUnwrapsFences(t *testing.T) {
	plain := "func main() {\n    print(1)\n}\n"
	if got := extractSource(plain); got != plain {
		t.Errorf("plain reply changed: %q", got)
	}
	fenced := "```chip\n" + plain + "```\n"
	if got := extractSource(fenced); strings.TrimSpace(got) != strings.TrimSpace(plain) {
		t.Errorf("fenced reply not unwrapped: %q", got)
	}
}
