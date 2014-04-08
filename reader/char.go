//
//  CHP/READER. Asynchronously read and return characters via a concurrent channel.
//
//    Jack Spirou
//    15 March 14
//

// Package reader provides a reader for UTF-8-encoded text.
// It takes an io.Reader providing the file source, then returns the
// characters of that source via a channel.  If the first character
// in the source is a UTF-8 encoded byte order mark (BOM), it is discarded.
//
// Basic usage pattern:
//
//      src := bufio.NewReader(file)
//      chrs := reader.NewReaderChannel(src)
//      for ch != range chrs {
//      	// do something with ch
//      }
//

package reader

import "unicode/utf8"

// A Reader implements reading of Unicode characters and tokens from an io.Reader.
type Char struct {
	chr rune
	err error
}

func NewChar(chr rune, err error) *Char {
	return &Char{chr: chr, err: err}
}

func (c *Char) ReadRune() (r rune, size int, err error) {
	return c.chr, utf8.RuneLen(c.chr), nil
}

func (c *Char) Rune() rune {
	return c.chr
}

func (c *Char) Error() error {
	return c.err
}
