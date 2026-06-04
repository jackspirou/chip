package chip_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/jackspirou/chip"
)

// Run streams: a top-level statement may call a function defined later in the
// source (the batch compile path would not run top-level statements at all).
func TestRunStreamsForwardRef(t *testing.T) {
	src := []byte("print(f())\nfunc f() int { return 42 }\n")
	var out bytes.Buffer
	if err := chip.Run(src, &out); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := out.String(); got != "42\n" {
		t.Fatalf("stdout = %q, want %q", got, "42\n")
	}
}

// A streaming type error surfaces from Run as a public *chip.Error.
func TestRunReturnsTypeError(t *testing.T) {
	var out bytes.Buffer
	err := chip.Run([]byte("func f() int { return \"x\" }\n"), &out)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	var e *chip.Error
	if !errors.As(err, &e) {
		t.Fatalf("error type = %T, want *chip.Error", err)
	}
	if !strings.Contains(e.Error(), "cannot return string as int") {
		t.Fatalf("error = %v, want it to contain %q", e, "cannot return string as int")
	}
}
