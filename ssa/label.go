package ssa

import (
	"log"
	"strconv"
	"unicode"
	"unicode/utf8"
)

var count = 0 // the number of labels created

// Label describes a SSA label. Label is modeled off the MIPS instruction set.
type Label struct {
	count  int    // Uniquifying counter.
	length int    // Max PREFIX length.
	name   string // What to print.
	placed bool   // Has label been placed?
	prefix string // Default prefix.
}

// NewLabel returns a new label whos name consists of a prefix followed by
// some digits which make it unique.
//
// Only the first length characters of the prefix are used, and the prefix must
// not end with a digit. If no prefix is given, then the default is used.
//
// Label is initially not initally placed.
func NewLabel(prefix string) *Label {

	l := &Label{
		count:  count,
		length: 5,
		prefix: "label",
	}

	if utf8.RuneCountInString(prefix) > utf8.RuneCountInString(l.prefix) {
		l.prefix = prefix[:utf8.RuneCountInString(prefix)-1]
	}

	lastRune, _ := utf8.DecodeLastRuneInString(l.prefix)

	if prefix == "main" {
		l.name = "main"
		return l
	}

	if unicode.IsDigit(lastRune) {
		log.Fatal("label ended with a digit")
	}

	l.name = l.prefix + strconv.Itoa(l.count)
	count++
	return l
}

// NewBlankLabel returns a new blank Label object.
func NewBlankLabel() *Label {

	l := &Label{
		count:  count,
		length: 5,
		prefix: "label",
		name:   "label" + strconv.Itoa(count),
	}

	count++

	return l
}

// Place records that a label has been placed. If the label has already been
// placed previously an error will be logged, and execution will hault.
func (l *Label) Place() {
	if l.placed {
		log.Fatalf("'%s' cannot be placed twice", l)
	}
	l.placed = true
}

// String impliments the fmt.Stringer interface.
func (l *Label) String() string {
	return l.name
}
