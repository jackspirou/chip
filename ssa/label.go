package ssa

import (
	"strconv"
	"unicode"
	"unicode/utf8"
)

var count = 0

type Label struct {
	count  int    // Uniquifying counter.
	length int    // Max PREFIX length.
	name   string // What to print.
	placed bool   // Has label been placed?
	prefix string // Default prefix.
}

// Constructors. Make a new LABEL whose NAME consists of a PREFIX, followed by
// some digits that make it unique. Only the first LENGTH characters of PREFIX
// are used, and PREFIX must not end with a digit. If no PREFIX is given, then
// use the default. The LABEL is initially not PLACED.

func NewLabel(prefix string) *Label {
	l := new(Label)
	l.count = count
	l.length = 5
	l.prefix = "label"
	l.placed = false

	if utf8.RuneCountInString(prefix) > utf8.RuneCountInString(l.prefix) {
		l.prefix = prefix[:utf8.RuneCountInString(prefix)-1]
	}

	lastRune, _ := utf8.DecodeLastRuneInString(l.prefix)

	if prefix == "main" {
		l.name = "main"
	} else if unicode.IsDigit(lastRune) {
		panic("Label ended with a digit.")
	} else {
		l.name = l.prefix + strconv.Itoa(l.count)
		count++
	}
	return l
}

func NewBlankLabel() *Label {
	l := new(Label)
	l.count = count
	l.length = 5
	l.prefix = "label"
	l.placed = false
	l.name = l.prefix + strconv.Itoa(l.count)
	count++
	return l
}

// PLACE. If this LABEL has been PLACED, then throw an exception. Otherwise
// we record that this LABEL is now PLACED.

func (this *Label) Place() {
	if this.placed {
		panic("'" + this.name + "' was placed twice.")
	} else {
		this.placed = true
	}
}

// STRING. Convert this LABEL to a STRING for printing.
func (this *Label) String() string {
	return this.name
}
