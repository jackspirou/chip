package check_test

import (
	"testing"

	"github.com/jackspirou/chip/internal/check"
	"github.com/jackspirou/chip/internal/diag"
)

// findUndefined returns the "undefined: <name>" error from checking src.
func findUndefined(t *testing.T, src, name string) check.Error {
	t.Helper()
	err := checkSrc(t, src)
	el, ok := err.(check.ErrorList)
	if !ok {
		t.Fatalf("error is %T (%v), want check.ErrorList", err, err)
	}
	want := "undefined: " + name
	for _, e := range el {
		if e.Msg == want {
			return e
		}
	}
	t.Fatalf("no %q error in %v", want, el)
	return check.Error{}
}

// A near-miss undefined name carries a grounded "did you mean" suggestion: a
// Maybe-applicability edit replacing the misspelled identifier's exact byte span
// with the in-scope name (A3).
func TestDidYouMeanNearMiss(t *testing.T) {
	e := findUndefined(t, `package main
func main() {
    count := 1
    print(coont)
}`, "coont")

	if len(e.Suggestions) != 1 {
		t.Fatalf("got %d suggestions, want 1: %+v", len(e.Suggestions), e.Suggestions)
	}
	s := e.Suggestions[0]
	if s.Message != "did you mean count?" {
		t.Errorf("message = %q, want %q", s.Message, "did you mean count?")
	}
	if s.Applicability != diag.Maybe {
		t.Errorf("applicability = %q, want %q", s.Applicability, diag.Maybe)
	}
	if len(s.Edit) != 1 {
		t.Fatalf("got %d edits, want 1", len(s.Edit))
	}
	ed := s.Edit[0]
	if ed.NewText != "count" {
		t.Errorf("edit.NewText = %q, want %q", ed.NewText, "count")
	}
	if ed.EndOffset-ed.Offset != len("coont") {
		t.Errorf("edit span = [%d,%d) = %d bytes, want %d (the %q token)",
			ed.Offset, ed.EndOffset, ed.EndOffset-ed.Offset, len("coont"), "coont")
	}
}

// A name with no near neighbor in scope gets no suggestion: a hint is offered
// only when grounded, never guessed (D14). "undef" is too far from every in-scope
// name, so the error stands alone.
func TestNoSuggestionWhenNothingClose(t *testing.T) {
	e := findUndefined(t, `package main
func main() {
    print(undef)
}`, "undef")
	if len(e.Suggestions) != 0 {
		t.Fatalf("got %d suggestions, want 0: %+v", len(e.Suggestions), e.Suggestions)
	}
}
