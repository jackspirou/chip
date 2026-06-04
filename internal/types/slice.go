package types

// Slice is the type of a dynamically-sized sequence, []Elem.
type Slice struct {
	Elem Type
}

func (Slice) typ() {}

func (s Slice) String() string {
	if s.Elem == nil {
		return "[]invalid"
	}
	return "[]" + s.Elem.String()
}
