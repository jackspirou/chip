// Package eval is chip's mini harness: a deterministic integration rig that drives
// the chip CLI end to end, and an evaluation rig that loops a model over a corpus of
// broken→fixed programs to measure how an agent's repair loop converges (plan §5.3).
//
// The two jobs share one corpus and one notion of "solved". The integration rig
// (integration_test.go) drives the built binary — stream in, validate, cap,
// capture — and asserts exit codes, the JSON Result envelope, and captured bytes.
// The eval rig (this file) feeds a Model the diagnostics from a failing program,
// rendered in a chosen Profile, and counts turns-to-green and tokens-to-green — the
// numbers that ground per-model feedback tuning (§5.5). The model is an interface:
// CI runs a deterministic ScriptedModel (no network, no key); a live run uses the
// Anthropic driver (anthropic.go), exercised only when a key is present.
package eval

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jackspirou/chip/internal/analyze"
	"github.com/jackspirou/chip/internal/diag"
	"github.com/jackspirou/chip/internal/stream"
)

// Profile selects how the harness renders a failing program's diagnostics back to
// the model between turns — the variable the eval rig exists to compare. The values
// map to chip's renderers; the per-model "llm" / "llm:<model>" profile (§5.5)
// renders the grounded LLM view — agent diagnostics plus the catalog's repair hints
// — mirroring the CLI.
type Profile string

const (
	ProfileHuman Profile = "human" // the caret view a person reads
	ProfileAgent Profile = "agent" // terse, greppable, coded lines
	ProfileLLM   Profile = "llm"   // agent lines plus grounded catalog repair hints
	ProfileJSON  Profile = "json"  // the Result envelope
)

// AllProfiles is the set the rig compares by default.
var AllProfiles = []Profile{ProfileHuman, ProfileAgent, ProfileLLM, ProfileJSON}

// rendererFor maps a profile to the diag.Renderer that formats feedback, mirroring
// the CLI's resolution: "llm" and any "llm:<model>" tuning render the grounded LLM
// view; an otherwise-unknown profile renders as agent.
func rendererFor(p Profile) diag.Renderer {
	switch {
	case p == ProfileHuman:
		return diag.Human{}
	case p == ProfileJSON:
		return diag.JSON{}
	case p == ProfileLLM || strings.HasPrefix(string(p), "llm:"):
		return diag.LLM{}
	default:
		return diag.Agent{}
	}
}

// FeedbackMode selects which of chip's two diagnostic disciplines the rig feeds
// back between turns — the run=fail-fast vs check=collect-all split (chip's D3).
// The streaming run reports only the first fault; the batch check reports every
// static error at once. The two lead a model to repair differently, so the §5.5
// per-model feedback comparison fixes the mode it measures under.
type FeedbackMode int

const (
	// FeedbackCollectAll feeds the batch checker's diagnostics — every static error at
	// once (`chip check`), falling back to the run's own fault when the checker is
	// silent (the D3 boundary, e.g. a division by zero, which only faults at run). It
	// is the zero value, so a Config that does not set Feedback measures under
	// collect-all, unchanged.
	FeedbackCollectAll FeedbackMode = iota
	// FeedbackStream feeds the streaming run's first fault only (`chip run`), the
	// fail-fast view an agent streaming programs into chip actually sees. One error at
	// a time is where grounded per-model feedback earns its keep: the model must
	// generalize the fix from a single flagged instance instead of being handed every
	// occurrence, so a terse code it misreads by analogy costs a whole extra turn.
	FeedbackStream
)

// Config tunes a repair run.
type Config struct {
	Profile  Profile      // how diagnostics are rendered back to the model
	MaxTurns int          // give up after this many attempts (the plan caps at ~3)
	Contract string       // optional preamble prepended to the first prompt (e.g. `chip prompt`)
	Feedback FeedbackMode // which diagnostic discipline to feed back (default: collect-all)
}

// Case is one repair scenario: a broken program and the contract for "fixed". Want
// is the exact stdout the fixed program must print; Fixed is a canonical solution
// the integration rig checks and the scripted model can replay.
type Case struct {
	Name   string
	Broken string
	Fixed  string
	Want   string
}

// Outcome records how one model's repair run on one case went.
type Outcome struct {
	Case   string
	Solved bool
	Turns  int // attempts made (1..Config.MaxTurns)
	Tokens int // total tokens the model reported across those attempts
}

// Solve loops the model on one case until the program it returns runs clean and
// prints Want, or MaxTurns attempts are spent. The broken program's own
// diagnostics seed the first prompt; each later turn sends the previous attempt
// and its diagnostics, rendered in cfg.Profile. It returns the turns and tokens
// spent — the rig's per-case measurement.
func Solve(ctx context.Context, m Model, c Case, cfg Config) Outcome {
	out := Outcome{Case: c.Name}
	maxTurns := cfg.MaxTurns
	if maxTurns < 1 {
		maxTurns = 3
	}
	candidate := c.Broken
	// Seed turn 1 with what chip already says about the broken program: a real
	// agent loop starts from the diagnostics, not a bare "here is some code".
	_, ds, _ := validate(ctx, candidate, c.Want, cfg.Feedback)
	feedback := render(cfg.Profile, candidate, ds)
	for turn := 1; turn <= maxTurns; turn++ {
		out.Turns = turn
		reply, tokens, err := m.Complete(ctx, buildPrompt(cfg, candidate, feedback))
		out.Tokens += tokens
		if err != nil {
			return out
		}
		candidate = extractSource(reply)
		solved, nextDiags, _ := validate(ctx, candidate, c.Want, cfg.Feedback)
		if solved {
			out.Solved = true
			return out
		}
		feedback = render(cfg.Profile, candidate, nextDiags)
	}
	return out
}

// validate runs the candidate under caps and reports whether it is solved (ran
// clean and printed want) along with the diagnostics to feed back if not. Solved is
// always decided by the run — chip's runtime is the ground truth — so the verdict is
// identical across feedback modes; only the diagnostics fed back differ.
//
// The feedback mode picks which discipline supplies those diagnostics. Under
// FeedbackCollectAll the batch checker leads (every static error at once), falling
// back to the run's own fault when the checker is silent (the D3 boundary, e.g. a
// division by zero). Under FeedbackStream the run's first fault leads (the fail-fast
// view), falling back to the checker only if the run produced none. Either way a
// program that runs clean but prints the wrong thing gets a synthetic output-mismatch
// diagnostic, so the loop never sends empty feedback for a real failure.
func validate(ctx context.Context, src, want string, feedback FeedbackMode) (bool, []diag.Diagnostic, string) {
	var buf bytes.Buffer
	runErr := runUnderCaps(ctx, src, &buf)
	got := buf.String()
	if runErr == nil && (want == "" || got == want) {
		return true, nil, got
	}

	checkDiags := func() []diag.Diagnostic { return analyze.Source([]byte(src)).Diagnostics }
	runDiags := func() []diag.Diagnostic {
		if runErr != nil {
			return diag.From(runErr)
		}
		return nil
	}

	var ds []diag.Diagnostic
	if feedback == FeedbackStream {
		if ds = runDiags(); len(ds) == 0 {
			ds = checkDiags()
		}
	} else {
		if ds = checkDiags(); len(ds) == 0 {
			ds = runDiags()
		}
	}
	if len(ds) == 0 {
		ds = []diag.Diagnostic{outputMismatch(want, got)}
	}
	return false, ds, got
}

// runUnderCaps runs a candidate program with the host sealed off and bounded
// resources: a model can emit a runaway or a program that tries to import, so deny
// host imports (bundled stdlib still resolves) and cap steps, output, and
// wall-clock. The step budget makes a runaway terminate deterministically.
func runUnderCaps(ctx context.Context, src string, out io.Writer) error {
	cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return stream.RunWithLimits(cctx, strings.NewReader(src), out, stream.DenyLoader(), stream.Limits{
		Steps:    5_000_000,
		OutBytes: 1 << 16,
	})
}

// render formats the diagnostics in the chosen profile, exactly as `chip` would,
// so the feedback the model sees matches what the CLI emits. It returns "" when
// there is nothing to report.
func render(p Profile, src string, ds []diag.Diagnostic) string {
	if len(ds) == 0 {
		return ""
	}
	var buf bytes.Buffer
	_ = rendererFor(p).Render(&buf, diag.Source{Name: "<candidate>", Bytes: []byte(src)}, diag.WithCodes(ds))
	return buf.String()
}

// outputMismatch is the diagnostic for a program that checks and runs clean but
// prints the wrong thing — the one failure mode static analysis can't catch.
func outputMismatch(want, got string) diag.Diagnostic {
	return diag.Diagnostic{
		Severity: diag.SeverityError,
		Phase:    diag.PhaseRuntime,
		Message:  fmt.Sprintf("program printed %q but expected %q", got, want),
		Primary:  diag.Primary{IsPrimary: true, Line: 1, Column: 1},
	}
}

// buildPrompt assembles the message sent to the model: an optional contract
// preamble, the instruction, the current program, and what chip reported about it.
func buildPrompt(cfg Config, program, feedback string) string {
	var b strings.Builder
	if cfg.Contract != "" {
		b.WriteString(cfg.Contract)
		b.WriteString("\n\n")
	}
	b.WriteString("Fix this chip program. Return only the corrected chip source — no prose, no Markdown fences.\n\n")
	b.WriteString("Program:\n")
	b.WriteString(program)
	if !strings.HasSuffix(program, "\n") {
		b.WriteByte('\n')
	}
	if feedback != "" {
		b.WriteString("\nchip reported:\n")
		b.WriteString(feedback)
	}
	return b.String()
}

// extractSource pulls chip source out of a model reply, unwrapping a single
// Markdown code fence (```` ``` ```` or ```` ```chip ````) if the model added one.
// Unfenced replies are returned unchanged so exact source bytes survive.
func extractSource(reply string) string {
	s := strings.TrimSpace(reply)
	if !strings.HasPrefix(s, "```") {
		return reply
	}
	if nl := strings.IndexByte(s, '\n'); nl >= 0 {
		s = s[nl+1:] // drop the opening ``` / ```chip line
	}
	if i := strings.LastIndex(s, "```"); i >= 0 {
		s = s[:i] // drop the closing fence
	}
	return s
}

// RunCorpus solves every case with a fresh model from newModel (so a stateful or
// scripted model gets a clean slate per case) and returns the outcomes in corpus
// order.
func RunCorpus(ctx context.Context, newModel func(Case) Model, cases []Case, cfg Config) []Outcome {
	outcomes := make([]Outcome, 0, len(cases))
	for _, c := range cases {
		outcomes = append(outcomes, Solve(ctx, newModel(c), c, cfg))
	}
	return outcomes
}

// Summary aggregates a profile's outcomes into one row of the results table.
type Summary struct {
	Profile     Profile
	Model       string
	Cases       int
	Solved      int
	AvgTurns    float64 // mean turns over the solved cases (0 if none solved)
	TotalTokens int     // tokens spent across all cases
}

// Summarize reduces a profile's outcomes to a row: how many cases went green, the
// mean turns-to-green, and the total tokens spent.
func Summarize(profile Profile, model string, outcomes []Outcome) Summary {
	s := Summary{Profile: profile, Model: model, Cases: len(outcomes)}
	turns := 0
	for _, o := range outcomes {
		s.TotalTokens += o.Tokens
		if o.Solved {
			s.Solved++
			turns += o.Turns
		}
	}
	if s.Solved > 0 {
		s.AvgTurns = float64(turns) / float64(s.Solved)
	}
	return s
}

// WriteTable prints the turns-/tokens-to-green table: one row per profile, with how
// many cases went green, the mean turns-to-green, and the total tokens spent.
func WriteTable(w io.Writer, rows []Summary) {
	fmt.Fprintf(w, "%-8s %-18s %-8s %-12s %s\n", "profile", "model", "solved", "turns/green", "tokens")
	for _, r := range rows {
		fmt.Fprintf(w, "%-8s %-18s %-8s %-12.2f %d\n",
			r.Profile, r.Model, fmt.Sprintf("%d/%d", r.Solved, r.Cases), r.AvgTurns, r.TotalTokens)
	}
}
