package reader

import (
	"strings"
	"testing"
)

func TestReader(t *testing.T) {
	src := strings.NewReader(content)
	rdr := New(src)
	for {
		ch, err := rdr.Read()
		if err != nil {
			t.Errorf("%s: %c", err, ch)
		}
		if ch == EOF {
			break
		}
	}
}

const content = `package main

import "fmt"

func main() {
  fmt.Println("hello world")
}`
