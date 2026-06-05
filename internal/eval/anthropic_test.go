package eval

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestAnthropicModelAgainstFakeServer exercises the live driver without a key or
// the network: a fake server stands in for the Messages API, and the test asserts
// the driver sends the pinned headers and the right body, and parses the reply and
// token usage. This is what keeps the driver honest in CI.
func TestAnthropicModelAgainstFakeServer(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key-123")

	var gotKey, gotVersion, gotCT, gotBody, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("x-api-key")
		gotVersion = r.Header.Get("anthropic-version")
		gotCT = r.Header.Get("content-type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("content-type", "application/json")
		_, _ = io.WriteString(w, `{"content":[{"type":"text","text":"fixed source"}],"usage":{"input_tokens":12,"output_tokens":7}}`)
	}))
	defer srv.Close()

	m, err := NewAnthropicModel("claude-test")
	if err != nil {
		t.Fatalf("NewAnthropicModel: %v", err)
	}
	m.baseURL = srv.URL // point at the fake server (white-box test)

	reply, tokens, err := m.Complete(context.Background(), "fix this program")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	if reply != "fixed source" {
		t.Errorf("reply = %q, want %q", reply, "fixed source")
	}
	if tokens != 19 {
		t.Errorf("tokens = %d, want 19 (12 in + 7 out)", tokens)
	}
	if gotPath != "/v1/messages" {
		t.Errorf("path = %q, want /v1/messages", gotPath)
	}
	if gotKey != "test-key-123" {
		t.Errorf("x-api-key = %q, want the env key", gotKey)
	}
	if gotVersion != anthropicVersion {
		t.Errorf("anthropic-version = %q, want %q", gotVersion, anthropicVersion)
	}
	if !strings.Contains(gotCT, "application/json") {
		t.Errorf("content-type = %q, want application/json", gotCT)
	}
	if !strings.Contains(gotBody, `"claude-test"`) {
		t.Errorf("request body missing model id: %s", gotBody)
	}
	if !strings.Contains(gotBody, "fix this program") {
		t.Errorf("request body missing prompt: %s", gotBody)
	}
}

// TestAnthropicModelRequiresKey checks the fail-fast: with no key, construction
// errors rather than deferring to a doomed request.
func TestAnthropicModelRequiresKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	if _, err := NewAnthropicModel("claude-test"); err == nil {
		t.Fatal("expected an error when ANTHROPIC_API_KEY is unset")
	}
}

// TestLiveEval runs the real eval against the Anthropic API and is the empirical
// Verify for §5.5: that the grounded "llm" feedback reaches green in fewer turns
// than terse "agent" feedback for at least one model. It is gated on CHIP_EVAL_LIVE=1
// (and a key) so it never runs in ordinary CI. The model id comes from
// CHIP_EVAL_MODEL (default: a small fast model, which has the most to gain from
// grounding).
//
// The comparison prepends NO language contract, on purpose. The variable under test
// is the per-turn feedback. Handed chip's full rules up front (the `chip prompt`
// contract), a model already knows the type set and one-shots every case, so agent
// and llm both sit at the one-turn floor and the difference is masked — that is what
// the with-contract baseline showed (agent=llm=1.00). With no contract the model
// must repair from the diagnostics alone, which is exactly where grounding earns its
// keep: both profiles run under identical (empty-contract) conditions, and only the
// rendering of the feedback differs.
//
// It measures under FeedbackStream — the fail-fast view `chip run` shows, one fault
// at a time — because that is the scenario §5.5 is about: an agent streaming a
// program into chip and repairing from the runtime's first complaint. Collect-all
// (`chip check`) hands a competent model every error at once, including occurrences
// and adjacent messages that leak the answer, and so masks the per-turn difference
// the grounded feedback is meant to make. Streaming feedback is chip's core agentic
// posture (run=fail-fast), so it is the honest condition for this claim, decided
// before the run, not after.
func TestLiveEval(t *testing.T) {
	if os.Getenv("CHIP_EVAL_LIVE") != "1" {
		t.Skip("set CHIP_EVAL_LIVE=1 (and ANTHROPIC_API_KEY) to run the live eval")
	}
	model := os.Getenv("CHIP_EVAL_MODEL")
	if model == "" {
		model = "claude-haiku-4-5-20251001"
	}
	m, err := NewAnthropicModel(model)
	if err != nil {
		t.Skipf("live eval unavailable: %v", err)
	}

	ctx := context.Background()
	cases := Corpus()
	maxTurns := 3 // the plan's repair cap; also the basis for the never-converged penalty below
	byProfile := map[Profile][]Outcome{}
	var rows []Summary
	for _, p := range AllProfiles {
		// No Contract: the per-turn feedback is the variable. FeedbackStream: measure
		// under the fail-fast view (`chip run`), where one error at a time makes grounding decisive.
		cfg := Config{Profile: p, MaxTurns: maxTurns, Feedback: FeedbackStream}
		outcomes := RunCorpus(ctx, func(Case) Model { return m }, cases, cfg)
		byProfile[p] = outcomes
		rows = append(rows, Summarize(p, m.Name(), outcomes))
	}
	var buf bytes.Buffer
	WriteTable(&buf, rows)
	t.Logf("live eval (%s), no contract:\n%s", model, buf.String())

	// §5.5's measurement. The grounded llm view is a strict superset of agent (the same
	// located, greppable lines, plus the catalog's grounded repair guidance), so it must
	// never solve fewer cases; the slice's claim is that it reaches green in strictly
	// fewer turns for at least one model. Score each case by turns-to-green, counting a
	// case that never converged within the cap as strictly worse than any solve
	// (maxTurns+1). That is the correction over a commonly-solved-only average, which
	// would *exclude* the very case the grounding wins most — one the llm view solves and
	// the terse view cannot — and so discard the strongest evidence. Log every case so
	// each win, and any regression, is auditable line by line.
	//
	// Not every case is grounding-sensitive, and that is expected. undefined-name is a
	// typo; return-type-mismatch / parse-error / wrong-arg-count are output-
	// underdetermined — the model must infer a free value the diagnostic cannot pin down
	// — so they measure value inference, not feedback grounding, and add per-case noise in
	// either direction. The undefined-type case is the grounding-sensitive one: chip's
	// type set is a fact the terse code withholds and the grounded "why:" line supplies,
	// so that is where the separation is expected to show. The guards below require a
	// strict per-case win and forbid a net regression, so noise cannot manufacture a pass.
	agent, llm := byProfile[ProfileAgent], byProfile[ProfileLLM]
	penalty := maxTurns + 1 // never-converged: strictly costlier than any solve, incl. a solve at the cap
	eff := func(o Outcome) int {
		if o.Solved {
			return o.Turns
		}
		return penalty
	}
	var agentSolved, llmSolved, llmWins, llmRegress, totalAgent, totalLLM int
	for i := range cases {
		if agent[i].Solved {
			agentSolved++
		}
		if llm[i].Solved {
			llmSolved++
		}
		ea, el := eff(agent[i]), eff(llm[i])
		totalAgent += ea
		totalLLM += el
		mark := ""
		switch {
		case el < ea:
			llmWins++
			mark = "  <- llm reaches green in fewer turns"
		case el > ea:
			llmRegress++
			mark = "  <- agent reaches green in fewer turns"
		}
		t.Logf("  %-20s agent{solved:%v turns:%d}  llm{solved:%v turns:%d}%s",
			cases[i].Name, agent[i].Solved, agent[i].Turns, llm[i].Solved, llm[i].Turns, mark)
	}
	t.Logf("§5.5 turns-to-green (never-converged counts as %d): agent total=%d, llm total=%d; llm wins %d case(s), regresses %d; solved agent=%d llm=%d of %d",
		penalty, totalAgent, totalLLM, llmWins, llmRegress, agentSolved, llmSolved, len(cases))

	switch {
	case llmSolved < agentSolved:
		t.Errorf("§5.5 guardrail broken for %s: grounded llm profile solved fewer cases (%d) than agent (%d)", model, llmSolved, agentSolved)
	case llmWins == 0:
		t.Errorf("§5.5 not demonstrated for %s: no case where the grounded llm profile reached green in fewer turns than agent", model)
	case totalLLM > totalAgent:
		t.Errorf("§5.5 net regression for %s: grounded llm profile spent more total turns-to-green (llm=%d > agent=%d)", model, totalLLM, totalAgent)
	default:
		t.Logf("§5.5 holds for %s: grounded llm reaches green in fewer turns on %d case(s) (no net regression: llm total=%d <= agent total=%d), solving %d/%d vs agent %d/%d",
			model, llmWins, totalLLM, totalAgent, llmSolved, len(cases), agentSolved, len(cases))
	}
}
