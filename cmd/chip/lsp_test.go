package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/jackspirou/chip/internal/diag"
)

// §5.6 Verify: a didSave publishes diagnostics matching `chip check --json` for
// the same source. We take `chip check --json` as ground truth, remap its 1-based
// positions to LSP's 0-based coordinates, and assert the server publishes exactly
// that — so editor squiggles can never drift from the CLI's diagnostics.
func TestLspPublishesDiagnosticsMatchingCheck(t *testing.T) {
	const src = "func main() {\n\tprint(undef)\n}\n"
	const uri = "file:///broken.chp"

	// Ground truth: the diagnostics `chip check --json` reports for this source.
	var checkOut, checkErr bytes.Buffer
	run([]string{"check", "--json", "-e", src}, nil, &checkOut, &checkErr)
	var want diag.Result
	if err := json.Unmarshal(checkErr.Bytes(), &want); err != nil {
		t.Fatalf("check --json did not produce the JSON envelope: %v\n%s", err, checkErr.String())
	}
	if len(want.Diagnostics) == 0 {
		t.Fatal("expected check to report at least one diagnostic for the broken source")
	}

	// Drive the server through the open/save lifecycle, then exit.
	in := concatFrames(
		lspFrame(t, lspRequest("initialize", 1, map[string]any{})),
		lspFrame(t, lspNotify("textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{"uri": uri, "text": src},
		})),
		lspFrame(t, lspNotify("textDocument/didSave", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"text":         src,
		})),
		lspFrame(t, lspNotify("exit", nil)),
	)
	var out bytes.Buffer
	if err := cmdLsp([]string{"--stdio"}, bytes.NewReader(in), &out); err != nil {
		t.Fatalf("cmdLsp returned an error: %v", err)
	}

	pubs := publishNotifications(t, out.Bytes())
	if len(pubs) < 2 {
		t.Fatalf("expected publishDiagnostics for both didOpen and didSave, got %d", len(pubs))
	}
	save := pubs[len(pubs)-1] // the didSave's publish is the last one
	if save.uri != uri {
		t.Fatalf("publish uri = %q, want %q", save.uri, uri)
	}

	// What the published diagnostics must equal: check's diagnostics remapped to LSP.
	wantLSP := make([]lspDiagnostic, 0, len(want.Diagnostics))
	for _, d := range want.Diagnostics {
		wantLSP = append(wantLSP, lspDiagnostic{
			Range: lspRange{
				Start: lspPosition{Line: zeroBased(d.Primary.Line), Character: zeroBased(d.Primary.Column)},
				End:   lspPosition{Line: zeroBased(d.Primary.EndLine), Character: zeroBased(d.Primary.EndColumn)},
			},
			Severity: lspSeverity(d.Severity),
			Code:     d.Code,
			Source:   "chip",
			Message:  d.Message,
		})
	}
	if !reflect.DeepEqual(save.ds, wantLSP) {
		t.Fatalf("published diagnostics do not match check --json\n got: %+v\nwant: %+v", save.ds, wantLSP)
	}
}

// initialize advertises a diagnostics-only server: open/close tracking, full-text
// sync (change: 1), and save with the document text included.
func TestLspInitializeAdvertisesCapabilities(t *testing.T) {
	in := concatFrames(
		lspFrame(t, lspRequest("initialize", 7, map[string]any{})),
		lspFrame(t, lspNotify("exit", nil)),
	)
	var out bytes.Buffer
	if err := cmdLsp([]string{"--stdio"}, bytes.NewReader(in), &out); err != nil {
		t.Fatalf("cmdLsp: %v", err)
	}
	frames := decodeLspFrames(t, out.Bytes())
	if len(frames) == 0 {
		t.Fatal("server sent no initialize response")
	}
	var resp struct {
		ID     int `json:"id"`
		Result struct {
			Capabilities struct {
				TextDocumentSync struct {
					OpenClose bool `json:"openClose"`
					Change    int  `json:"change"`
					Save      struct {
						IncludeText bool `json:"includeText"`
					} `json:"save"`
				} `json:"textDocumentSync"`
			} `json:"capabilities"`
			ServerInfo struct {
				Name string `json:"name"`
			} `json:"serverInfo"`
		} `json:"result"`
	}
	if err := json.Unmarshal(frames[0], &resp); err != nil {
		t.Fatalf("initialize response not decodable: %v\n%s", err, frames[0])
	}
	if resp.ID != 7 {
		t.Fatalf("response id = %d, want 7 (must echo the request id)", resp.ID)
	}
	sync := resp.Result.Capabilities.TextDocumentSync
	if !sync.OpenClose || sync.Change != 1 || !sync.Save.IncludeText {
		t.Fatalf("capabilities not advertised correctly: %+v", sync)
	}
	if resp.Result.ServerInfo.Name != "chip" {
		t.Fatalf("serverInfo.name = %q, want chip", resp.Result.ServerInfo.Name)
	}
}

// A valid program publishes an empty diagnostics array (never null), so an editor
// clears any stale squiggles.
func TestLspCleanSourcePublishesEmpty(t *testing.T) {
	const src = "func main() {\n\tprint(6 * 7)\n}\n"
	const uri = "file:///clean.chp"
	in := concatFrames(
		lspFrame(t, lspRequest("initialize", 1, map[string]any{})),
		lspFrame(t, lspNotify("textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{"uri": uri, "text": src},
		})),
		lspFrame(t, lspNotify("exit", nil)),
	)
	var out bytes.Buffer
	if err := cmdLsp([]string{"--stdio"}, bytes.NewReader(in), &out); err != nil {
		t.Fatalf("cmdLsp: %v", err)
	}
	pubs := publishNotifications(t, out.Bytes())
	if len(pubs) != 1 {
		t.Fatalf("expected exactly one publish, got %d", len(pubs))
	}
	if len(pubs[0].ds) != 0 {
		t.Fatalf("clean source published %d diagnostics, want 0", len(pubs[0].ds))
	}
	// And the wire form must be [] not null, so clients clear prior squiggles.
	if !bytes.Contains(out.Bytes(), []byte(`"diagnostics":[]`)) {
		t.Fatalf("clean publish must carry an empty array, got:\n%s", out.String())
	}
}

// didChange replaces the document; a save-then-fix cycle reflects the new text:
// a broken open publishes a diagnostic, a change to valid source clears it.
func TestLspDidChangeRepublishes(t *testing.T) {
	const uri = "file:///edit.chp"
	in := concatFrames(
		lspFrame(t, lspRequest("initialize", 1, map[string]any{})),
		lspFrame(t, lspNotify("textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{"uri": uri, "text": "func main() {\n\tprint(undef)\n}\n"},
		})),
		lspFrame(t, lspNotify("textDocument/didChange", map[string]any{
			"textDocument":   map[string]any{"uri": uri},
			"contentChanges": []any{map[string]any{"text": "func main() {\n\tprint(7)\n}\n"}},
		})),
		lspFrame(t, lspNotify("exit", nil)),
	)
	var out bytes.Buffer
	if err := cmdLsp([]string{"--stdio"}, bytes.NewReader(in), &out); err != nil {
		t.Fatalf("cmdLsp: %v", err)
	}
	pubs := publishNotifications(t, out.Bytes())
	if len(pubs) != 2 {
		t.Fatalf("expected two publishes (open, change), got %d", len(pubs))
	}
	if len(pubs[0].ds) == 0 {
		t.Fatal("the broken open should have published a diagnostic")
	}
	if len(pubs[1].ds) != 0 {
		t.Fatalf("the fixed change should clear diagnostics, got %d", len(pubs[1].ds))
	}
}

// didClose clears the document's diagnostics with an empty publish.
func TestLspDidCloseClearsDiagnostics(t *testing.T) {
	const uri = "file:///closing.chp"
	in := concatFrames(
		lspFrame(t, lspRequest("initialize", 1, map[string]any{})),
		lspFrame(t, lspNotify("textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{"uri": uri, "text": "func main() {\n\tprint(undef)\n}\n"},
		})),
		lspFrame(t, lspNotify("textDocument/didClose", map[string]any{
			"textDocument": map[string]any{"uri": uri},
		})),
		lspFrame(t, lspNotify("exit", nil)),
	)
	var out bytes.Buffer
	if err := cmdLsp([]string{"--stdio"}, bytes.NewReader(in), &out); err != nil {
		t.Fatalf("cmdLsp: %v", err)
	}
	pubs := publishNotifications(t, out.Bytes())
	if len(pubs) != 2 {
		t.Fatalf("expected two publishes (open, close), got %d", len(pubs))
	}
	if len(pubs[1].ds) != 0 {
		t.Fatalf("close should clear diagnostics, got %d", len(pubs[1].ds))
	}
}

// shutdown returns an explicit null result and exit ends the read loop cleanly.
func TestLspShutdownThenExit(t *testing.T) {
	in := concatFrames(
		lspFrame(t, lspRequest("shutdown", 99, nil)),
		lspFrame(t, lspNotify("exit", nil)),
	)
	var out bytes.Buffer
	if err := cmdLsp([]string{"--stdio"}, bytes.NewReader(in), &out); err != nil {
		t.Fatalf("cmdLsp should return nil after exit, got %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte(`"result":null`)) {
		t.Fatalf("shutdown must reply with a null result, got:\n%s", out.String())
	}
}

// An unknown request gets a -32601 reply so the client is never left waiting.
func TestLspUnknownRequestReplies(t *testing.T) {
	in := concatFrames(
		lspFrame(t, lspRequest("textDocument/hover", 5, map[string]any{})),
		lspFrame(t, lspNotify("exit", nil)),
	)
	var out bytes.Buffer
	if err := cmdLsp([]string{"--stdio"}, bytes.NewReader(in), &out); err != nil {
		t.Fatalf("cmdLsp: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte(`"code":-32601`)) {
		t.Fatalf("unknown method must get method-not-found, got:\n%s", out.String())
	}
}

// An unknown transport flag is a usage error (exit 64), like the other commands.
func TestLspRejectsUnknownFlag(t *testing.T) {
	var out bytes.Buffer
	err := cmdLsp([]string{"--tcp"}, strings.NewReader(""), &out)
	var ue *usageError
	if !errors.As(err, &ue) {
		t.Fatalf("want a usageError for an unknown flag, got %v", err)
	}
}

// --- framing helpers -------------------------------------------------------

// lspRequest and lspNotify build the JSON-RPC message maps the client sends.
func lspRequest(method string, id int, params any) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}
}

func lspNotify(method string, params any) map[string]any {
	m := map[string]any{"jsonrpc": "2.0", "method": method}
	if params != nil {
		m["params"] = params
	}
	return m
}

// lspFrame encodes one message in the LSP wire format (Content-Length header,
// blank line, JSON body).
func lspFrame(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal frame: %v", err)
	}
	return fmt.Appendf(nil, "Content-Length: %d\r\n\r\n%s", len(b), b)
}

func concatFrames(frames ...[]byte) []byte {
	return bytes.Join(frames, nil)
}

// decodeLspFrames splits the server's framed output into the raw JSON bodies,
// mirroring readRPC's header parsing.
func decodeLspFrames(t *testing.T, data []byte) []json.RawMessage {
	t.Helper()
	r := bufio.NewReader(bytes.NewReader(data))
	var out []json.RawMessage
	for {
		length := -1
		for {
			line, err := r.ReadString('\n')
			if err == io.EOF && line == "" {
				return out // clean end at a frame boundary
			}
			if err != nil {
				t.Fatalf("reading header: %v", err)
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				break
			}
			if name, val, ok := strings.Cut(line, ":"); ok && strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
				n, perr := strconv.Atoi(strings.TrimSpace(val))
				if perr != nil {
					t.Fatalf("bad Content-Length %q", val)
				}
				length = n
			}
		}
		if length < 0 {
			t.Fatal("frame had no Content-Length header")
		}
		body := make([]byte, length)
		if _, err := io.ReadFull(r, body); err != nil {
			t.Fatalf("reading body: %v", err)
		}
		out = append(out, append(json.RawMessage(nil), body...))
	}
}

// publish pairs a publishDiagnostics notification's uri with its diagnostics.
type publish struct {
	uri string
	ds  []lspDiagnostic
}

// publishNotifications returns every textDocument/publishDiagnostics notification
// the server sent, in order, as (uri, diagnostics) pairs.
func publishNotifications(t *testing.T, data []byte) []publish {
	t.Helper()
	var out []publish
	for _, raw := range decodeLspFrames(t, data) {
		var head struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if json.Unmarshal(raw, &head) != nil || head.Method != "textDocument/publishDiagnostics" {
			continue
		}
		var params struct {
			URI         string          `json:"uri"`
			Diagnostics []lspDiagnostic `json:"diagnostics"`
		}
		if err := json.Unmarshal(head.Params, &params); err != nil {
			t.Fatalf("publishDiagnostics params not decodable: %v", err)
		}
		out = append(out, publish{uri: params.URI, ds: params.Diagnostics})
	}
	return out
}
