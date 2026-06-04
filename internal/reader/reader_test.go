package reader

import (
	"bytes"
	"strings"
	"testing"
)

const ReadingText = `package main

import "fmt"

func main() {
  // p.enter()("hello world")
}`

func TestReaderReadingText(t *testing.T) {
	src := strings.NewReader(ReadingText)
	r := New(src)
	for {
		ch, err := r.Next()
		if err != nil {
			t.Errorf("%s: %c", err, ch)
		}
		if ch == EOF {
			break
		}
	}
}

const SkippingBOMText = string(BOM) + `package main

import "fmt"

func main() {
  // p.enter()("hello world")
}`

func TestReaderSkippingBOM(t *testing.T) {
	src := strings.NewReader(SkippingBOMText)
	r := New(src)
	for {
		ch, err := r.Next()
		if err != nil {
			t.Errorf("%s: %c", err, ch)
		}
		if ch == BOM {
			t.Errorf("%s: %c", "Reader Error: BOM character not skipped ", ch)
		}
		if ch == EOF {
			break
		}
	}
}

func TestReaderNoText(t *testing.T) {
	var b []byte
	src := bytes.NewReader(b)
	r := New(src)
	ch, err := r.Next()
	if err != nil {
		t.Errorf("%s: %c", err, ch)
	}
	if ch != EOF {
		t.Errorf("%s: %c", "Reader Error: Source with missing text did not render an EOF character ", ch)
	}
}
