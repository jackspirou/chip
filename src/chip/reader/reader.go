// Package reader provides an implimention to read UTF-8 characters from a source.
//
package reader

import (
	"errors"
	"io"
	"unicode/utf8"
)

const (

	// EOF represents an end of file flag
	EOF = -1

	// buffer should be at least utf8.UTFMax (4) bytes
	bufLen = 1024
)

// Reader represents a reader to read Unicode characters from a source.
type Reader struct {

	// input source
	src io.Reader

	// source buffer
	srcBuf [bufLen + 1]byte // +1 for sentinel for common case of s.next()
	srcPos int              // reading position (srcBuf index)
	srcEnd int              // source end (srcBuf index)

	// source position byte offset of srcBuf[0]
	srcBufOffset int

	// one character look-ahead, char before current srcPos
	ch rune
}

// New takes an io.Reader and returns a new Reader.
func New(src io.Reader) *Reader {

	// create a new reader
	r := &Reader{

		// set source
		src: src,

		// default positions to 0
		// srcPos: 0,
		// srcEnd: 0,

		// initialize source position to 0
		// srcBufOffset: 0,

		// initialize one character look-ahead
		ch: EOF, // no char read yet
	}

	// initialize source buffer
	// (the first call to next() will fill it by calling src.Read)
	r.srcBuf[0] = utf8.RuneSelf // sentinel

	return r
}

// Read returns the next Unicode character in the source and advances
// the Reader position. It returns EOF if the reader's position is at the last
// character of the source.
func (r *Reader) Read() (rune, error) {

	// get the next char by peaking ahead
	ch, err := r.Peek()

	// error check
	if err != nil {
		return EOF, err
	}

	// advance to the next char
	char, err := r.next()

	// set the current char equal to the next char
	r.ch = char

	// return the peak ahead char
	return ch, err
}

// Peek returns the next Unicode character in the source without advancing
// the Reader. It returns EOF if the scanner's position is at the last
// character of the source.
func (r *Reader) Peek() (rune, error) {

	// check if this is the first character
	if r.ch < 0 {

		// call next only for the very first character
		char, err := r.next()

		// error check
		if err != nil {
			return EOF, err
		}

		// set reader char
		r.ch = char

		// check for BOM
		if r.ch == '\uFEFF' {

			// ignore BOM
			char, err := r.next()

			// error check
			if err != nil {
				return EOF, err
			}

			// set reader char
			r.ch = char
		}
	}

	return r.ch, nil
}

// next reads and returns the next Unicode character. It is designed such
// that only a minimal amount of work needs to be done in the common ASCII
// case (one test to check for both ASCII and end-of-buffer, and one test
// to check for newlines).
func (r *Reader) next() (rune, error) {

	// get the char from the buffer
	ch, width := rune(r.srcBuf[r.srcPos]), 1

	// compare ch to utf8.RuneSelf
	if ch >= utf8.RuneSelf {

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
		ch = rune(r.srcBuf[r.srcPos])
		if ch >= utf8.RuneSelf {

			// uncommon case: not ASCII
			ch, width = utf8.DecodeRune(r.srcBuf[r.srcPos:r.srcEnd])
			if ch == utf8.RuneError && width == 1 {

				// advance for correct error position
				r.srcPos += width
				return EOF, errors.New("illegal UTF-8 encoding")
			}
		}
	}

	// advance the position
	r.srcPos += width

	// special situations
	if ch == 0 {

		// for compatibility with other tools
		return EOF, errors.New("illegal character NUL")
	}

	return ch, nil
}
