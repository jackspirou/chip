package diag

import (
	"fmt"
	"sort"
)

// ApplyEdits returns src with every edit applied. Edits are byte-range splices
// (Offset, EndOffset, NewText); an insertion is an empty range. The edits must not
// overlap — overlapping edits have no well-defined combined result, so ApplyEdits
// refuses them rather than guess. This is the apply primitive behind `chip fix`
// (D6: the diagnostic carries the edits, one applier realizes them).
//
// Edits are applied highest-offset-first so each splice leaves the offsets of the
// not-yet-applied (lower) edits valid — the bytes before an edit are never moved.
// The input slice is never mutated; the result is a fresh slice.
func ApplyEdits(src []byte, edits []Edit) ([]byte, error) {
	if len(edits) == 0 {
		return src, nil
	}
	sorted := append([]Edit(nil), edits...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Offset != sorted[j].Offset {
			return sorted[i].Offset > sorted[j].Offset
		}
		return sorted[i].EndOffset > sorted[j].EndOffset
	})

	out := src
	prevLow := len(src) // lowest offset touched by an already-applied edit
	for _, e := range sorted {
		if e.Offset < 0 || e.EndOffset < e.Offset || e.EndOffset > len(src) {
			return nil, fmt.Errorf("edit out of bounds: [%d,%d) in %d-byte source", e.Offset, e.EndOffset, len(src))
		}
		if e.EndOffset > prevLow {
			return nil, fmt.Errorf("overlapping edits near offset %d", e.Offset)
		}
		var b []byte
		b = append(b, out[:e.Offset]...)
		b = append(b, e.NewText...)
		b = append(b, out[e.EndOffset:]...)
		out = b
		prevLow = e.Offset
	}
	return out, nil
}
