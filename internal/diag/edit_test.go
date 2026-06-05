package diag

import "testing"

// TestApplyEditsSingle applies one replacement and checks the splice lands exactly
// on the byte range, leaving the surrounding bytes untouched.
func TestApplyEditsSingle(t *testing.T) {
	src := []byte("hello world")
	out, err := ApplyEdits(src, []Edit{{Offset: 6, EndOffset: 11, NewText: "chip"}})
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}
	if string(out) != "hello chip" {
		t.Errorf("out = %q, want %q", out, "hello chip")
	}
}

// TestApplyEditsMultipleOutOfOrder applies several non-overlapping edits handed in
// low-to-high order: the highest-offset-first application must still place each one
// correctly, since a naive left-to-right splice would shift later offsets.
func TestApplyEditsMultipleOutOfOrder(t *testing.T) {
	src := []byte("aaa bbb ccc")
	edits := []Edit{
		{Offset: 0, EndOffset: 3, NewText: "X"},    // aaa -> X
		{Offset: 4, EndOffset: 7, NewText: "YY"},   // bbb -> YY
		{Offset: 8, EndOffset: 11, NewText: "ZZZ"}, // ccc -> ZZZ
	}
	out, err := ApplyEdits(src, edits)
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}
	if string(out) != "X YY ZZZ" {
		t.Errorf("out = %q, want %q", out, "X YY ZZZ")
	}
}

// TestApplyEditsInsertion uses an empty range to insert without removing anything.
func TestApplyEditsInsertion(t *testing.T) {
	src := []byte("ac")
	out, err := ApplyEdits(src, []Edit{{Offset: 1, EndOffset: 1, NewText: "b"}})
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}
	if string(out) != "abc" {
		t.Errorf("out = %q, want %q", out, "abc")
	}
}

// TestApplyEditsDeletion removes a range (empty NewText) — the LNT004 dead-code
// shape: splice out [start,end) and keep the rest verbatim.
func TestApplyEditsDeletion(t *testing.T) {
	src := []byte("keep DROP keep")
	out, err := ApplyEdits(src, []Edit{{Offset: 4, EndOffset: 9, NewText: ""}})
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}
	if string(out) != "keep keep" {
		t.Errorf("out = %q, want %q", out, "keep keep")
	}
}

// TestApplyEditsEmpty returns the source unchanged when there are no edits.
func TestApplyEditsEmpty(t *testing.T) {
	src := []byte("untouched")
	out, err := ApplyEdits(src, nil)
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}
	if string(out) != "untouched" {
		t.Errorf("out = %q, want %q", out, "untouched")
	}
}

// TestApplyEditsOverlap rejects two edits whose ranges intersect, since their
// combined result is undefined.
func TestApplyEditsOverlap(t *testing.T) {
	src := []byte("hello world")
	edits := []Edit{
		{Offset: 0, EndOffset: 6, NewText: "X"},
		{Offset: 3, EndOffset: 9, NewText: "Y"},
	}
	if _, err := ApplyEdits(src, edits); err == nil {
		t.Fatal("expected an error for overlapping edits")
	}
}

// TestApplyEditsOutOfBounds rejects an edit whose range falls outside the source.
func TestApplyEditsOutOfBounds(t *testing.T) {
	src := []byte("short")
	cases := []Edit{
		{Offset: -1, EndOffset: 2, NewText: "x"}, // negative start
		{Offset: 2, EndOffset: 99, NewText: "x"}, // end past EOF
		{Offset: 4, EndOffset: 2, NewText: "x"},  // end before start
	}
	for _, e := range cases {
		if _, err := ApplyEdits(src, []Edit{e}); err == nil {
			t.Errorf("expected an error for out-of-bounds edit %+v", e)
		}
	}
}

// TestApplyEditsDoesNotMutateInput proves the source slice is never written
// through: a caller can apply edits and still hold the original bytes.
func TestApplyEditsDoesNotMutateInput(t *testing.T) {
	src := []byte("hello world")
	orig := string(src)
	if _, err := ApplyEdits(src, []Edit{{Offset: 0, EndOffset: 5, NewText: "HELLO"}}); err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}
	if string(src) != orig {
		t.Errorf("input mutated: %q, want %q", src, orig)
	}
}

// TestApplyEditsAdjacent allows edits that touch at a boundary (end of one equals
// start of the next): they do not overlap, so both apply.
func TestApplyEditsAdjacent(t *testing.T) {
	src := []byte("abcd")
	edits := []Edit{
		{Offset: 0, EndOffset: 2, NewText: "X"}, // ab -> X
		{Offset: 2, EndOffset: 4, NewText: "Y"}, // cd -> Y
	}
	out, err := ApplyEdits(src, edits)
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}
	if string(out) != "XY" {
		t.Errorf("out = %q, want %q", out, "XY")
	}
}
