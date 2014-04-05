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
	"fmt"
	"io"
	"os"
	"unicode/utf8"
	)

	const bufLen = 1024 // at least utf8.UTFMax
	const EOF = -1

	// A Reader implements reading of Unicode characters and tokens from an io.Reader.
	type Reader struct {
		// Channel of runes
		chars chan *Char

		// Input
		src io.Reader

		// Source buffer
		srcBuf [bufLen + 1]byte // +1 for sentinel for common case of s.next()
		srcPos int              // reading position (srcBuf index)
		srcEnd int              // source end (srcBuf index)

		// Source position
		srcBufOffset int // byte offset of srcBuf[0] in source
		line         int // line count
		column       int // character count
		lastLineLen  int // length of last line in characters (for correct column reporting)
		lastCharLen  int // length of last character in bytes

		// One character look-ahead
		chr rune // character before current srcPos

		// Last error
		err error
	}

	// NewReader initializes a Scanner with a new source and returns s.
	// Error is set to nil, ErrorCount is set to 0, Mode is set to GoTokens.
	func NewReader(src io.Reader) *Reader {
		r := &Reader{
			chars: make(chan *Char),
			src:  src,

			srcPos: 0,
			srcEnd: 0,

			// initialize source position
			srcBufOffset: 0,
			line:         1,
			column:       0,
			lastLineLen:  0,
			lastCharLen:  0,

			// initialize one character look-ahead
			chr: -1, // no char read yet

			// No errors yet
			err: nil,
		}

		// initialize source buffer
		// (the first call to next() will fill it by calling src.Read)
		r.srcBuf[0] = utf8.RuneSelf // sentinel

		return r
	}

	func (r *Reader) GoRead() chan *Char {
		go r.run()
		return r.chars
	}

	func (r *Reader) run() {
		chr := r.read()
		for chr != EOF {
			r.chars <- NewChar(chr, r.err)
			chr = r.read()
		}
		close(r.chars)
	}

	// next reads and returns the next Unicode character. It is designed such
	// that only a minimal amount of work needs to be done in the common ASCII
	// case (one test to check for both ASCII and end-of-buffer, and one test
	// to check for newlines).
	func (r *Reader) next() rune {
		chr, width := rune(r.srcBuf[r.srcPos]), 1

		if chr >= utf8.RuneSelf {
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
						if r.lastCharLen > 0 {
							// previous character was not EOF
							r.column++
						}
						r.lastCharLen = 0
						return EOF
					}
					if err != io.EOF {
						r.err = err
					}
					// If err == EOF, we won't be getting more
					// bytes; break to avoid infinite loop. If
					// err is something else, we don't know if
					// we can get more bytes; thus also break.
					break
				}
			}
			// at least one byte
			chr = rune(r.srcBuf[r.srcPos])
			if chr >= utf8.RuneSelf {
				// uncommon case: not ASCII
				chr, width = utf8.DecodeRune(r.srcBuf[r.srcPos:r.srcEnd])
				if chr == utf8.RuneError && width == 1 {
					// advance for correct error position
					r.srcPos += width
					r.lastCharLen = width
					r.column++
					r.err = errors.New("illegal UTF-8 encoding")
					return chr
				}
			}
		}

		// advance
		r.srcPos += width
		r.lastCharLen = width
		r.column++

		// special situations
		switch chr {
			case 0:
			// for compatibility with other tools
			r.err = errors.New("illegal character NUL")
			case '\n':
			r.line++
			r.lastLineLen = r.column
			r.column = 0
		}

		return chr
	}

	// Next reads and returns the next Unicode character.
	// It returns EOF at the end of the source. It reports
	// a read error by calling s.Error, if not nil; otherwise
	// it prints an error message to os.Stderr. Next does not
	// update the Scanner's Position field; use Pos() to
	// get the current position.
	func (r *Reader) read() rune {
		chr := r.peek()
		r.chr = r.next()
		return chr
	}

	// Peek returns the next Unicode character in the source without advancing
	// the Reader. It returns EOF if the scanner's position is at the last
	// character of the source.
	func (r *Reader) peek() rune {
		if r.chr < 0 {
			// this code is only run for the very first character
			r.chr = r.next()
			if r.chr == '\uFEFF' {
				r.chr = r.next() // ignore BOM
			}
		}
		return r.chr
	}

	func (r *Reader) PrintErr() {
		fmt.Fprintf(os.Stderr, "%s: \n", r.err.Error())
	}

	func (r *Reader) PrintErrMsg(msg string) {
		fmt.Fprintf(os.Stderr, "%s: \n", msg)
	}
