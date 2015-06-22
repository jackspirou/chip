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
//      r := reader.NewReader(src)
//			chrs := r.GoRead()
//      for chr := range chrs {
//				if(ch.Error() == nill) {
//					// do something with chr
//				}else{
//					// error out
//				}
//      }
//

package reader

import (
	"errors"
	"io"
	"unicode/utf8"
)

const bufLen = 1024 // at least utf8.UTFMax
const EOF = -1

// A Reader implements reading of Unicode characters and tokens from an io.Reader.
type Reader struct {
	// Channel of runes
	chrs chan rune

	// Channel of errors
	errs chan error

	// Input
	src io.Reader

	// Source buffer
	srcBuf [bufLen + 1]byte // +1 for sentinel for common case of s.next()
	srcPos int              // reading position (srcBuf index)
	srcEnd int              // source end (srcBuf index)

	// Source position
	srcBufOffset int // byte offset of srcBuf[0] in source

	// One character look-ahead
	ch rune // character before current srcPos
}

// NewReader initializes a Scanner with a new source and returns s.
// Error is set to nil, ErrorCount is set to 0, Mode is set to GoTokens.
func NewReader(src io.Reader) *Reader {
	r := &Reader{
		chrs: make(chan rune),
		errs: make(chan error),
		src:  src,

		srcPos: 0,
		srcEnd: 0,

		// initialize source position
		srcBufOffset: 0,

		// initialize one character look-ahead
		ch: EOF, // no char read yet
	}

	// initialize source buffer
	// (the first call to next() will fill it by calling src.Read)
	r.srcBuf[0] = utf8.RuneSelf // sentinel

	return r
}

func (r *Reader) GoRead() (chan rune, chan error) {
	go r.run()
	return r.chrs, r.errs
}

func (r *Reader) run() {
	ch := r.read()
	for ch != EOF {
		r.chrs <- ch
		ch = r.read()
	}
	r.chrs <- ch
	close(r.chrs)
}

// next reads and returns the next Unicode character. It is designed such
// that only a minimal amount of work needs to be done in the common ASCII
// case (one test to check for both ASCII and end-of-buffer, and one test
// to check for newlines).
func (r *Reader) next() rune {
	ch, width := rune(r.srcBuf[r.srcPos]), 1

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
					return EOF
				}
				if err != io.EOF {
					r.errs <- err
					return EOF
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
				r.errs <- errors.New("illegal UTF-8 encoding")
				return EOF
			}
		}
	}

	// advance
	r.srcPos += width

	// special situations
	if ch == 0 {
		// for compatibility with other tools
		r.errs <- errors.New("illegal character NUL")
		return EOF
	}

	return ch
}

// Next reads and returns the next Unicode character.
// It returns EOF at the end of the source. It reports
// a read error by calling s.Error, if not nil; otherwise
// it prints an error message to os.Stderr. Next does not
// update the Scanner's Position field; use Pos() to
// get the current position.
func (r *Reader) read() rune {
	ch := r.peek()
	r.ch = r.next()
	return ch
}

// Peek returns the next Unicode character in the source without advancing
// the Reader. It returns EOF if the scanner's position is at the last
// character of the source.
func (r *Reader) peek() rune {
	if r.ch < 0 {
		// this code is only run for the very first character
		r.ch = r.next()
		if r.ch == '\uFEFF' {
			r.ch = r.next() // ignore BOM
		}
	}
	return r.ch
}
