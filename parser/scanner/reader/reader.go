// Package reader reads characters from a UTF-8 io.Reader source.
package reader

import (
	"errors"
	"io"
	"unicode/utf8"
)

const (

	// EOF represents the end of file rune.
	EOF = -1

	// BOM runes are ignored.
	BOM = '\uFEFF'

	// buffer should be at least utf8.UTFMax (4) bytes.
	bufLen = 1024
)

// Reader describes a reader that reads Unicode characters from an io.Reader
// source.
type Reader struct {
	src          io.Reader
	srcBuf       [bufLen + 1]byte // +1 for sentinel for common case of s.next()
	srcPos       int              // reading position (srcBuf index)
	srcEnd       int              // source end (srcBuf index)
	srcBufOffset int              // source position byte offset of srcBuf[0]
	char         rune             // one char look-ahead, before current srcPos
}

// New returns a new Reader object.
func New(src io.Reader) *Reader {

	r := &Reader{src: src, char: EOF}

	// initialize source buffer
	// (the first call to next() will fill it by calling src.Read)
	r.srcBuf[0] = utf8.RuneSelf // sentinel

	return r
}

// Read returns the next Unicode character in the source and advances
// the Reader position. It returns EOF if the reader's position is at the last
// character of the source.
func (r *Reader) Read() (rune, error) {

	char, err := r.Peek()
	if err != nil {
		return EOF, err
	}

	nextChar, err := r.next()

	// set the current char equal to the next char
	r.char = nextChar

	// return the char ahead char
	return char, err
}

// Peek returns the next Unicode character in the source without advancing
// the Reader. It returns EOF if the scanner's position is at the last
// character of the source.
func (r *Reader) Peek() (rune, error) {

	// if first char
	if r.char < 0 {

		char, err := r.next()
		if err != nil {
			return EOF, err
		}

		// set reader char
		r.char = char

		if r.char == BOM {

			// ignore BOM
			char, err := r.next()
			if err != nil {
				return EOF, err
			}

			// set reader char
			r.char = char
		}
	}

	return r.char, nil
}

// next reads and returns the next Unicode character. It is designed such that
// only a minimal amount of work needs to be done in the common ASCII case
// (one test to check for both ASCII and end-of-buffer, and one test to check
// for newlines).
func (r *Reader) next() (rune, error) {

	// char from buffer
	char, width := rune(r.srcBuf[r.srcPos]), 1

	// compare char to utf8.RuneSelf
	if char >= utf8.RuneSelf {

		// uncommon case: not ASCII or not enough bytes
		for r.srcPos+utf8.UTFMax > r.srcEnd && !utf8.FullRune(r.srcBuf[r.srcPos:r.srcEnd]) {

			// not enough bytes: read some more
			// move unread bytes to beginning of buffer
			copy(r.srcBuf[0:], r.srcBuf[r.srcPos:r.srcEnd])
			r.srcBufOffset += r.srcPos

			// read more bytes
			// (an io.Reader must return io.EOF when it reaches
			// the end of what it is reading - simply returning
			// n == 0 will make this loop retry forever; but the
			// error is in the reader implementation in that case)
			i := r.srcEnd - r.srcPos
			n, err := r.src.Read(r.srcBuf[i:bufLen])
			r.srcPos = 0
			r.srcEnd = i + n
			r.srcBuf[r.srcEnd] = utf8.RuneSelf // sentinel
			if err != nil {
				if r.srcEnd == 0 {
					return EOF, nil
				}
				if err != io.EOF {
					return EOF, err
				}

				// If err == EOF, we won't be getting more
				// bytes; break to avoid infinite loop. If
				// err is something else, we don't know if
				// we can get more bytes; thus also break.
				break
			}
		}

		// at least one byte
		char = rune(r.srcBuf[r.srcPos])
		if char >= utf8.RuneSelf {

			// uncommon case: not ASCII
			char, width = utf8.DecodeRune(r.srcBuf[r.srcPos:r.srcEnd])
			if char == utf8.RuneError && width == 1 {

				// advance for correct error position
				r.srcPos += width
				return EOF, errors.New("found illegal UTF-8 encoding")
			}
		}
	}

	// advance the position
	r.srcPos += width

	// special situations
	if char == 0 {

		// for compatibility with other tools
		return EOF, errors.New("found illegal character NUL")
	}

	return char, nil
}
