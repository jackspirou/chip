package stream

import (
	"bytes"
	"strings"
)

// Session is a persistent streaming evaluator driven one frame at a time. Like
// REPL, its engine tables carry over between Feed calls — a function defined in
// one frame is callable in the next, and a top-level variable keeps its value —
// but the caller owns the framing and reads each frame's output as a return
// value instead of from a fixed sink. It backs `chip session` (plan §5.7), the
// agentic notebook: a long-lived process an agent streams source into and reads
// structured events out of, building state turn by turn.
//
// A Session is single-goroutine, like the engine it wraps: Feed runs to
// completion before the next call. The same engine that powers Run and REPL
// powers it, so there is one evaluator and the session cannot drift from the
// language the file runner accepts.
type Session struct {
	e   *engine
	out *bytes.Buffer
}

// NewSession builds a session with the stdlib prelude loaded, resolving imports
// against the current directory (exactly as REPL does). The returned error is
// only a failure to load the prelude, which would also break every other run
// path; in practice it is nil.
func NewSession() (*Session, error) {
	buf := &bytes.Buffer{}
	e := newEngine(buf, DirLoader("."))
	if err := e.loadStd(); err != nil {
		return nil, err
	}
	return &Session{e: e, out: buf}, nil
}

// Feed runs one source chunk against the persistent engine and returns whatever
// the chunk printed. State persists for the next Feed: definitions register and
// top-level statements run in order, just as in REPL. A returned error is the
// chunk's first parse, type, or runtime fault (a Diagnoser, so the caller can
// render it structurally); any output produced before the fault is still
// returned, matching the engine's run-as-far-as-it-got streaming behavior.
func (s *Session) Feed(src string) (string, error) {
	s.out.Reset()
	err := s.e.feed(strings.NewReader(src))
	return s.out.String(), err
}
