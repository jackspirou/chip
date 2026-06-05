package main

// chip session is the agentic notebook (plan §5.7): a long-lived, full-duplex
// NDJSON conversation over the reusable streaming engine. The client writes one
// JSON frame per line on stdin — `{"source": "...", "id": <any>}` — and reads one
// JSON event per line on stdout. State carries across frames: a function or
// top-level variable defined in one frame is in scope for the next, because every
// frame feeds the SAME persistent engine (the same one behind `chip run` and
// `chip repl`), so the session can never accept a language the file runner would
// reject. It is single-goroutine and synchronous: a frame runs to completion, its
// events flush, then the next frame is read.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"github.com/jackspirou/chip/internal/diag"
	"github.com/jackspirou/chip/internal/stream"
)

// cmdSession runs the NDJSON session loop until stdin closes. It takes no
// arguments — frames arrive on stdin — so any argument is a usage error.
func cmdSession(args []string, stdin io.Reader, stdout io.Writer) error {
	for _, a := range args {
		return &usageError{msg: fmt.Sprintf("session takes no arguments, got %q", a)}
	}
	sess, err := stream.NewSession()
	if err != nil {
		return err
	}
	srv := &sessionServer{w: stdout, sess: sess}
	srv.emitReady()
	return srv.loop(stdin)
}

// sessionServer owns the framed output stream and the persistent session. A
// diagnostics-and-output protocol keeps no other state — the engine holds it all.
type sessionServer struct {
	w    io.Writer
	sess *stream.Session
}

// sessionFrame is one input line: a chunk of chip source plus an optional id the
// events for that frame echo back, so an async client can correlate them.
type sessionFrame struct {
	Source string          `json:"source"`
	ID     json.RawMessage `json:"id,omitempty"`
}

// sessionEvent is one output line. Event names: "ready" (one handshake at start),
// "output" (a frame's captured stdout, only when non-empty), "result" (a frame's
// terminal status — ok plus any diagnostics, the turn boundary), and "error" (a
// malformed frame, distinct from a chip diagnostic so protocol faults and program
// faults never blur).
type sessionEvent struct {
	Event       string            `json:"event"`
	ID          json.RawMessage   `json:"id,omitempty"`
	ChipVersion string            `json:"chipVersion,omitempty"`
	Text        string            `json:"text,omitempty"`
	OK          *bool             `json:"ok,omitempty"`
	Phase       diag.Phase        `json:"phase,omitempty"`
	Diagnostics []diag.Diagnostic `json:"diagnostics,omitempty"`
	Message     string            `json:"message,omitempty"`
}

// loop reads NDJSON frames until EOF, processing each in order. A blank line is
// skipped (as in the REPL); a line that is not a JSON object yields an error
// event and the loop continues, so one malformed frame never ends the session. A
// long line is read whole — bufio.Scanner's default token cap is lifted — so a
// big program in one frame is not silently truncated.
func (s *sessionServer) loop(stdin io.Reader) error {
	sc := bufio.NewScanner(stdin)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(trimSpaceBytes(line)) == 0 {
			continue
		}
		var frame sessionFrame
		if err := json.Unmarshal(line, &frame); err != nil {
			s.emit(sessionEvent{Event: "error", Message: "malformed frame: " + err.Error()})
			continue
		}
		s.run(frame)
	}
	return sc.Err()
}

// run feeds one frame's source through the persistent session and emits its
// events: the captured output first (when any), then the terminal result, which
// carries the frame's structured diagnostics when it faulted.
func (s *sessionServer) run(frame sessionFrame) {
	out, err := s.sess.Feed(frame.Source)
	if out != "" {
		s.emit(sessionEvent{Event: "output", ID: frame.ID, Text: out})
	}
	ok := err == nil
	ev := sessionEvent{Event: "result", ID: frame.ID, OK: &ok}
	if err != nil {
		// Same spine as `chip run --json`: convert the front-end error, attach codes,
		// and enrich against this frame's source so positions and snippets are present.
		ds := diag.Enrich(diag.WithCodes(diag.From(err)), diag.Source{Name: "<session>", Bytes: []byte(frame.Source)})
		if len(ds) > 0 {
			ev.Diagnostics = ds
			ev.Phase = ds[0].Phase
		} else {
			ev.Message = err.Error() // an error with no structured diagnostics
		}
	}
	s.emit(ev)
}

func (s *sessionServer) emitReady() {
	v, _, _ := versionInfo()
	s.emit(sessionEvent{Event: "ready", ChipVersion: v})
}

// emit writes one event as a single JSON line (NDJSON). A marshal failure is
// dropped rather than corrupting the stream; the event shapes are fixed structs,
// so it cannot happen in practice.
func (s *sessionServer) emit(ev sessionEvent) {
	b, err := json.Marshal(ev)
	if err != nil {
		return
	}
	fmt.Fprintf(s.w, "%s\n", b)
}

// trimSpaceBytes reports a line with leading/trailing ASCII whitespace removed,
// so a blank or all-spaces line is recognized without allocating a string.
func trimSpaceBytes(b []byte) []byte {
	start := 0
	for start < len(b) && asciiSpace(b[start]) {
		start++
	}
	end := len(b)
	for end > start && asciiSpace(b[end-1]) {
		end--
	}
	return b[start:end]
}

func asciiSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\v' || c == '\f'
}
