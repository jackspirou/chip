package eval

import "context"

// Model is the one thing the eval loop needs: turn a prompt into a reply, and say
// how many tokens that cost. Keeping it this small lets CI drive a deterministic
// ScriptedModel with no network or key, while a live run swaps in the Anthropic
// driver (anthropic.go) behind the same interface — the harness code never changes.
type Model interface {
	// Name identifies the model in the results table (e.g. "scripted", a model id).
	Name() string
	// Complete returns the model's reply to prompt and the tokens it reported
	// spending. A non-nil error ends the repair run for that case.
	Complete(ctx context.Context, prompt string) (reply string, tokens int, err error)
}

// ScriptedModel replays a fixed sequence of replies, charging a flat token cost per
// call. It is the deterministic stand-in that lets the whole harness — Solve,
// RunCorpus, the per-profile table — run in CI with no key and no network, so the
// rig's own logic is tested independently of any live model's behavior. Past the
// end of the script it repeats the last reply, so a model that "gives up" (keeps
// returning the same broken text) is easy to script.
type ScriptedModel struct {
	replies []string
	cost    int // tokens charged per Complete call
	calls   int // replies handed out so far
}

// NewScriptedModel returns a model that replies with each entry of replies in turn,
// reporting cost tokens per call. With no replies it always returns "".
func NewScriptedModel(replies []string, cost int) *ScriptedModel {
	return &ScriptedModel{replies: replies, cost: cost}
}

// Name reports the model identity for the results table.
func (*ScriptedModel) Name() string { return "scripted" }

// Complete hands back the next scripted reply (repeating the last once the script is
// spent) and charges the flat per-call cost. It never errors.
func (m *ScriptedModel) Complete(_ context.Context, _ string) (string, int, error) {
	if len(m.replies) == 0 {
		m.calls++
		return "", m.cost, nil
	}
	i := m.calls
	if i >= len(m.replies) {
		i = len(m.replies) - 1
	}
	m.calls++
	return m.replies[i], m.cost, nil
}
