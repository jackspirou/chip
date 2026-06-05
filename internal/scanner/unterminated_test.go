package scanner_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jackspirou/chip/internal/scanner"
	"github.com/jackspirou/chip/internal/token"
)

// TestUnterminatedStringTerminates locks the fix for an unbounded-memory hang in
// nextString. An unterminated string literal that reaches end of input used to
// spin forever: reader.EOF (-1) is neither '"' nor a newline, so the scan loop
// never exited and appended the U+FFFD replacement rune (three bytes) every
// iteration, growing the buffer without bound until the process exhausted memory.
// Scanning runs ahead of the execution step loop, so --timeout and --max-steps
// could not stop it — a single malformed literal in streamed input was a denial
// of service. The fix is the EOF guard in scanner.nextString; this test pins it.
//
// Each case runs under a watchdog so that, if the guard regresses, the test fails
// within seconds rather than hanging until the suite timeout while it exhausts
// memory (which is how this bug first surfaced — a probe input ate tens of GB).
func TestUnterminatedStringTerminates(t *testing.T) {
	cases := []struct{ name, src string }{
		{"plain at eof", `"abc`},
		{"bare quote at eof", `"`},
		{"escape-looking tail at eof", `"a\nb`},
		{"after leading tokens", `print("hi`},
		{"before newline", "\"hi\nbye"}, // the newline path already terminated; kept for parity
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			type result struct {
				tk  token.Token
				err error
			}
			done := make(chan result, 1)
			go func() {
				s, err := scanner.New(strings.NewReader(tc.src))
				if err != nil {
					done <- result{err: err}
					return
				}
				// Drain to the first terminal token. A correct scanner reaches an
				// ERROR (or EOF) in a handful of tokens for these tiny inputs; the
				// bounded loop is a second backstop against a different runaway.
				for range 1000 {
					tk := s.Next()
					if tk.Type == token.ERROR || tk.Type == token.EOF {
						done <- result{tk: tk}
						return
					}
				}
				done <- result{} // 1000 tokens without terminating: treated as a failure below
			}()

			select {
			case r := <-done:
				if r.err != nil {
					t.Fatalf("src %q: scanner.New: %v", tc.src, r.err)
				}
				if r.tk.Type != token.ERROR || !strings.Contains(r.tk.String(), "string has no closing quote") {
					t.Fatalf("src %q: got {type:%s lit:%q}, want ERROR %q",
						tc.src, r.tk.Type.String(), r.tk.String(), "string has no closing quote")
				}
			case <-time.After(3 * time.Second):
				t.Fatalf("src %q: scanner did not terminate within 3s — the nextString EOF guard has regressed (unbounded-memory hang)", tc.src)
			}
		})
	}
}
