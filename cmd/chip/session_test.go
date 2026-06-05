package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// §5.7 Verify: framed source-in / event-out with state across frames. A function
// defined in one frame is callable in the next, and that next frame's output
// comes back as an event — the whole point of the session over one-shot `run`.
func TestSessionStatePersistsAcrossFrames(t *testing.T) {
	in := strings.Join([]string{
		`{"source":"func double(n int) int { return n + n }\n"}`,
		`{"source":"print(double(21))\n"}`,
	}, "\n") + "\n"

	evs := driveSession(t, in)

	// First event is the ready handshake; every result is ok; the call frame's
	// output event carries 42.
	if len(evs) == 0 || evs[0].Event != "ready" {
		t.Fatalf("first event must be ready, got %+v", evs)
	}
	var sawOutput bool
	for _, e := range evs {
		switch e.Event {
		case "output":
			if e.Text == "42\n" {
				sawOutput = true
			}
		case "result":
			if e.OK == nil || !*e.OK {
				t.Fatalf("result not ok: %+v", e)
			}
		}
	}
	if !sawOutput {
		t.Fatalf("expected an output event carrying 42 from the second frame; events: %+v", evs)
	}
}

// A top-level variable bound in one frame keeps its value in a later frame.
func TestSessionTopLevelVarPersists(t *testing.T) {
	in := `{"source":"x := 21\n"}` + "\n" + `{"source":"print(x * 2)\n"}` + "\n"
	evs := driveSession(t, in)
	if got := lastOutput(evs); got != "42\n" {
		t.Fatalf("output across frames = %q, want %q", got, "42\n")
	}
}

// The session opens with a ready handshake carrying the chip version, so a client
// knows the process is live before sending source.
func TestSessionReadyHandshake(t *testing.T) {
	evs := driveSession(t, "") // no frames: just the handshake
	if len(evs) != 1 {
		t.Fatalf("want exactly the ready event, got %+v", evs)
	}
	if evs[0].Event != "ready" || evs[0].ChipVersion == "" {
		t.Fatalf("ready handshake malformed: %+v", evs[0])
	}
}

// A faulting frame reports structured diagnostics in its result (ok=false), the
// same codes the rest of the CLI emits, and the session survives for the next
// frame.
func TestSessionReportsDiagnosticsAndRecovers(t *testing.T) {
	in := `{"source":"print(undef)\n"}` + "\n" + `{"source":"print(6 * 7)\n"}` + "\n"
	evs := driveSession(t, in)

	var faulted *sessionEvent
	for i := range evs {
		if evs[i].Event == "result" && evs[i].OK != nil && !*evs[i].OK {
			faulted = &evs[i]
			break
		}
	}
	if faulted == nil {
		t.Fatalf("expected a failing result for the undefined name; events: %+v", evs)
	}
	if len(faulted.Diagnostics) == 0 || faulted.Diagnostics[0].Code != "NAM001" {
		t.Fatalf("want NAM001 in the failing result, got %+v", faulted.Diagnostics)
	}
	// The diagnostic is enriched (snippet present), matching the JSON envelope.
	if faulted.Diagnostics[0].Primary.Snippet == "" {
		t.Fatalf("diagnostic should be enriched with a snippet: %+v", faulted.Diagnostics[0])
	}
	// Recovery: the following valid frame still runs.
	if got := lastOutput(evs); got != "42\n" {
		t.Fatalf("session did not recover; last output = %q, want %q", got, "42\n")
	}
}

// A malformed (non-JSON) frame yields an error event and the session continues —
// one bad line never ends the conversation.
func TestSessionMalformedFrameContinues(t *testing.T) {
	in := "not json at all\n" + `{"source":"print(1)\n"}` + "\n"
	evs := driveSession(t, in)
	var sawError bool
	for _, e := range evs {
		if e.Event == "error" {
			sawError = true
		}
	}
	if !sawError {
		t.Fatalf("expected an error event for the malformed frame; events: %+v", evs)
	}
	if got := lastOutput(evs); got != "1\n" {
		t.Fatalf("valid frame after a malformed one did not run; last output = %q", got)
	}
}

// A frame's id is echoed on the events it produces, so an async client can
// correlate output and result with the source that caused them.
func TestSessionEchoesFrameID(t *testing.T) {
	in := `{"id":7,"source":"print(99)\n"}` + "\n"
	evs := driveSession(t, in)
	var sawIDOutput, sawIDResult bool
	for _, e := range evs {
		if string(e.ID) != "7" {
			continue
		}
		switch e.Event {
		case "output":
			sawIDOutput = true
		case "result":
			sawIDResult = true
		}
	}
	if !sawIDOutput || !sawIDResult {
		t.Fatalf("frame id 7 not echoed on output(%v) and result(%v); events: %+v", sawIDOutput, sawIDResult, evs)
	}
}

// session takes no arguments; a positional argument is a usage error (exit 64).
func TestSessionRejectsArgs(t *testing.T) {
	var out bytes.Buffer
	err := cmdSession([]string{"extra"}, strings.NewReader(""), &out)
	var ue *usageError
	if !errors.As(err, &ue) {
		t.Fatalf("want a usageError for a positional arg, got %v", err)
	}
}

// --- helpers ---------------------------------------------------------------

// driveSession runs cmdSession over the given NDJSON input and returns the
// decoded output events in order.
func driveSession(t *testing.T, in string) []sessionEvent {
	t.Helper()
	var out bytes.Buffer
	if err := cmdSession(nil, strings.NewReader(in), &out); err != nil {
		t.Fatalf("cmdSession: %v", err)
	}
	return decodeEvents(t, out.Bytes())
}

func decodeEvents(t *testing.T, data []byte) []sessionEvent {
	t.Helper()
	var evs []sessionEvent
	for line := range bytes.SplitSeq(bytes.TrimRight(data, "\n"), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var ev sessionEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			t.Fatalf("event not decodable: %v\n%s", err, line)
		}
		evs = append(evs, ev)
	}
	return evs
}

// lastOutput returns the text of the last output event, or "" if none.
func lastOutput(evs []sessionEvent) string {
	out := ""
	for _, e := range evs {
		if e.Event == "output" {
			out = e.Text
		}
	}
	return out
}
