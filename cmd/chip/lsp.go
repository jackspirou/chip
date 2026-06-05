package main

// chip lsp is a diagnostics-only Language Server (plan §5.5/5.6, D24). Editors
// that speak LSP (Cursor, Windsurf, Cline, VS Code) get native squiggles by
// launching `chip lsp --stdio`; on open/change/save it runs the SAME static
// analysis as `chip check` and publishes the result as LSP diagnostics, so the
// squiggles match `chip check --json` for the same source. It is strictly
// additive — a new subcommand over the existing diag spine — and single-goroutine
// (one read loop, synchronous handlers), like the rest of the engine.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/jackspirou/chip/internal/analyze"
	"github.com/jackspirou/chip/internal/diag"
)

// cmdLsp runs the language server over stdio until the client sends `exit` or the
// input stream closes. --stdio is accepted (and is the only transport) so editor
// launch configs that pass it work unchanged.
func cmdLsp(args []string, stdin io.Reader, stdout io.Writer) error {
	for _, a := range args {
		switch a {
		case "--stdio", "-stdio":
			// the only transport; accepted for editor launch configs
		default:
			return &usageError{msg: fmt.Sprintf("unknown flag for lsp: %s", a)}
		}
	}
	srv := &lspServer{w: stdout, docs: map[string][]byte{}}
	return srv.serve(stdin)
}

// lspServer holds the open documents (uri -> latest full text) and the framed
// output stream. A diagnostics-only server keeps no other state.
type lspServer struct {
	w    io.Writer
	docs map[string][]byte
}

func (s *lspServer) serve(stdin io.Reader) error {
	r := bufio.NewReader(stdin)
	for {
		msg, err := readRPC(r)
		if err == io.EOF {
			return nil // client closed the stream: clean shutdown
		}
		if err != nil {
			return err
		}
		if s.handle(msg) {
			return nil // `exit` notification
		}
	}
}

// rpcMessage is a decoded JSON-RPC 2.0 request or notification. ID is kept raw so
// it echoes back verbatim (a number or a string) in the response; a notification
// omits it (len(ID) == 0).
type rpcMessage struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// readRPC reads one Content-Length-framed JSON-RPC message: headers terminated by
// a blank line, then exactly that many body bytes. A clean EOF at a message
// boundary returns io.EOF; a truncated frame returns the read error.
func readRPC(r *bufio.Reader) (rpcMessage, error) {
	length := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			// EOF with no bytes buffered is a clean shutdown between messages.
			if err == io.EOF && line == "" {
				return rpcMessage{}, io.EOF
			}
			return rpcMessage{}, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break // blank line ends the headers
		}
		if name, val, ok := strings.Cut(line, ":"); ok && strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			n, perr := strconv.Atoi(strings.TrimSpace(val))
			if perr != nil {
				return rpcMessage{}, fmt.Errorf("lsp: bad Content-Length %q", val)
			}
			length = n
		}
	}
	if length < 0 {
		return rpcMessage{}, fmt.Errorf("lsp: message had no Content-Length header")
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(r, body); err != nil {
		return rpcMessage{}, err
	}
	var msg rpcMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return rpcMessage{}, fmt.Errorf("lsp: malformed message body: %w", err)
	}
	return msg, nil
}

// handle dispatches one message, returning true when the client asked to exit.
func (s *lspServer) handle(msg rpcMessage) (exit bool) {
	switch msg.Method {
	case "initialize":
		s.respond(msg.ID, initializeResult())
	case "shutdown":
		s.respond(msg.ID, nil)
	case "exit":
		return true
	case "textDocument/didOpen":
		s.didOpen(msg.Params)
	case "textDocument/didChange":
		s.didChange(msg.Params)
	case "textDocument/didSave":
		s.didSave(msg.Params)
	case "textDocument/didClose":
		s.didClose(msg.Params)
	default:
		// An unknown request must get a reply so the client is not left waiting; an
		// unknown notification (no id) is silently ignored, as the spec allows.
		if len(msg.ID) > 0 {
			s.respondError(msg.ID, -32601, "method not found: "+msg.Method)
		}
	}
	return false
}

// initializeResult advertises a diagnostics-only server: open/close tracking,
// full-text sync, and save with the document text included.
func initializeResult() map[string]any {
	v, _, _ := versionInfo()
	return map[string]any{
		"capabilities": map[string]any{
			"textDocumentSync": map[string]any{
				"openClose": true,
				"change":    1, // 1 = full document sync
				"save":      map[string]any{"includeText": true},
			},
		},
		"serverInfo": map[string]any{"name": "chip", "version": v},
	}
}

func (s *lspServer) didOpen(params json.RawMessage) {
	var p struct {
		TextDocument struct {
			URI  string `json:"uri"`
			Text string `json:"text"`
		} `json:"textDocument"`
	}
	if json.Unmarshal(params, &p) != nil {
		return
	}
	s.docs[p.TextDocument.URI] = []byte(p.TextDocument.Text)
	s.publish(p.TextDocument.URI)
}

func (s *lspServer) didChange(params json.RawMessage) {
	var p struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		ContentChanges []struct {
			Text string `json:"text"`
		} `json:"contentChanges"`
	}
	if json.Unmarshal(params, &p) != nil || len(p.ContentChanges) == 0 {
		return
	}
	// Full sync (change: 1): the last content change carries the whole document.
	s.docs[p.TextDocument.URI] = []byte(p.ContentChanges[len(p.ContentChanges)-1].Text)
	s.publish(p.TextDocument.URI)
}

func (s *lspServer) didSave(params json.RawMessage) {
	var p struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Text *string `json:"text"`
	}
	if json.Unmarshal(params, &p) != nil {
		return
	}
	if p.Text != nil { // includeText: prefer the saved bytes over the cache
		s.docs[p.TextDocument.URI] = []byte(*p.Text)
	}
	s.publish(p.TextDocument.URI)
}

func (s *lspServer) didClose(params json.RawMessage) {
	var p struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if json.Unmarshal(params, &p) != nil {
		return
	}
	delete(s.docs, p.TextDocument.URI)
	s.publishDiagnostics(p.TextDocument.URI, nil) // clear squiggles on close
}

// publish runs chip's static analysis over the document and sends the result.
func (s *lspServer) publish(uri string) {
	s.publishDiagnostics(uri, analyzeToLSP(s.docs[uri], uriToName(uri)))
}

// analyzeToLSP runs the SAME analysis as `chip check` (analyze.Source +
// WithCodes + Enrich) and maps each diagnostic to its LSP shape, so a didSave
// publishes exactly what `chip check --json` reports for the same source. Chip
// positions are 1-based; LSP positions are 0-based, so each is shifted by one.
func analyzeToLSP(src []byte, name string) []lspDiagnostic {
	rep := analyze.Source(src)
	ds := diag.Enrich(diag.WithCodes(rep.Diagnostics), diag.Source{Name: name, Bytes: src})
	out := make([]lspDiagnostic, 0, len(ds))
	for _, d := range ds {
		out = append(out, lspDiagnostic{
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
	return out
}

// LSP diagnostic shapes — the subset a diagnostics-only server emits.
type lspPosition struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type lspRange struct {
	Start lspPosition `json:"start"`
	End   lspPosition `json:"end"`
}

type lspDiagnostic struct {
	Range    lspRange `json:"range"`
	Severity int      `json:"severity,omitempty"`
	Code     string   `json:"code,omitempty"`
	Source   string   `json:"source,omitempty"`
	Message  string   `json:"message"`
}

// lspSeverity maps a chip severity to the LSP DiagnosticSeverity enum (1 = Error,
// 2 = Warning).
func lspSeverity(sev diag.Severity) int {
	if sev == diag.SeverityWarning {
		return 2
	}
	return 1
}

// zeroBased converts a 1-based chip line/column to a 0-based LSP coordinate,
// clamping the unset (0) value to 0.
func zeroBased(n int) int {
	if n > 0 {
		return n - 1
	}
	return 0
}

// uriToName turns a document URI into the file name diagnostics report; only the
// presentation fields use it (LSP diagnostics carry positions, not a path).
func uriToName(uri string) string {
	return strings.TrimPrefix(uri, "file://")
}

// JSON-RPC response/notification shapes. Result is not omitempty so a `shutdown`
// reply carries an explicit null result, as the protocol expects.
type rpcResult struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result"`
}

type rpcErrResult struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Error   *rpcError       `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

func (s *lspServer) respond(id json.RawMessage, result any) {
	s.writeRPC(rpcResult{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *lspServer) respondError(id json.RawMessage, code int, message string) {
	s.writeRPC(rpcErrResult{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}})
}

func (s *lspServer) publishDiagnostics(uri string, ds []lspDiagnostic) {
	if ds == nil {
		ds = []lspDiagnostic{} // stable [] (never null) so a clean save clears squiggles
	}
	s.writeRPC(rpcNotification{
		JSONRPC: "2.0",
		Method:  "textDocument/publishDiagnostics",
		Params:  map[string]any{"uri": uri, "diagnostics": ds},
	})
}

// writeRPC frames one message in the LSP wire format: a Content-Length header, a
// blank line, then the JSON body.
func (s *lspServer) writeRPC(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	fmt.Fprintf(s.w, "Content-Length: %d\r\n\r\n%s", len(b), b)
}
