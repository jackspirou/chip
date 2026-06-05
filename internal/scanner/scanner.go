// Package scanner scans tokens from a UTF-8 io.Reader.
package scanner

import (
	"bytes"
	"fmt"
	"io"
	"unicode"
	"unicode/utf8"

	"github.com/jackspirou/chip/internal/reader"
	"github.com/jackspirou/chip/internal/token"
)

// Scanner describes a scanner to scan tokens from a UTF-8 io.Reader source.
type Scanner struct {
	r     *reader.Reader
	char  rune      // current look-ahead character
	pos   token.Pos // position of the current character
	start token.Pos // position of the first character of the current token
}

// New takes an io.Reader and returns a new chip Scanner.
func New(src io.Reader) (*Scanner, error) {
	s := &Scanner{r: reader.New(src), pos: token.Pos{Line: 1, Column: 0}}

	// we must advance the scanner to first position
	if err := s.next(); err != nil {
		return nil, err
	}

	return s, nil
}

// Next returns the next token.Token in the source, with its byte-exact span
// recorded (see token.Token.End): End is the scanner position one past the
// token's final byte, so End().Offset-Pos().Offset is the token's byte length.
func (s *Scanner) Next() token.Token {
	tok := s.scan()
	return tok.WithEnd(s.pos)
}

// scan reads and returns the next token, positioned at its first character but
// without an end position; Next wraps it to record that.
func (s *Scanner) scan() token.Token {
	// skip any blank spaces
	if err := s.skipSpaces(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}

	// the token begins at the current character; remember where so every token
	// is positioned at its first character rather than the character after it.
	s.start = s.pos

	// letters yield identifiers
	if letter(s.char) {
		return s.nextIdentifier()
	}

	// digits yield numbers
	if digit(s.char) {
		return s.nextNumber(false)
	}

	// determine char
	switch s.char {
	case reader.EOF:
		return s.nextEOF()
	case '"':
		return s.nextString()
	case ':':
		return s.nextColon()
	case '.':
		return s.nextPeriod()
	case ',':
		return s.nextComma()
	case '(':
		return s.nextOpenParen()
	case ')':
		return s.nextCloseParen()
	case '[':
		return s.nextOpenBracket()
	case ']':
		return s.nextCloseBracket()
	case '{':
		return s.nextOpenBrace()
	case '}':
		return s.nextCloseBrace()
	case '+':
		return s.nextPlus()
	case '-':
		return s.nextDash()
	case '*':
		return s.nextStar()
	case '/':
		return s.nextSlash()
	case '%':
		return s.nextPercent()
	case '^':
		return s.nextCaret()
	case '<':
		return s.nextLess()
	case '>':
		return s.nextGreater()
	case '=':
		return s.nextEqual()
	case '!':
		return s.nextBang()
	case '&':
		return s.nextAmpersand()
	case '|':
		return s.nextPipe()
	default:
		return token.New(token.ERROR, fmt.Sprintf("unexpected '%c'", s.char), s.start)
	}
}

// next advances the scanners position and reads the next char.
// The only token the next method might return is a token.Error.
func (s *Scanner) next() error {
	char, err := s.r.Next()
	if err != nil && char != reader.EOF {
		return fmt.Errorf("bug: expected EOF got '%c', error: %s", char, err)
	}

	// Advance the byte offset past the rune we are leaving (the old s.char),
	// mirroring the per-rune Column increment below. The first advance (no rune
	// loaded yet, s.char == 0) and EOF (-1) contribute no source bytes.
	if s.char > 0 {
		s.pos.Offset += utf8.RuneLen(s.char)
	}

	s.char = char
	s.pos.Column++

	return nil
}

// skipSpaces skips all spaces until the next valid character.
func (s *Scanner) skipSpaces() error {
	// prevCR tracks whether the previous rune was a '\r', so a following '\n'
	// (the second half of a "\r\n" pair) is not counted as a second line break.
	prevCR := false
	for whitespace(s.char) || endOfLine(s.char) {
		if endOfLine(s.char) {
			// Treat "\n", a lone "\r", and "\r\n" each as a single line break.
			// Reset the column for every line-terminator rune so the first rune of
			// the next line lands at column 1, but bump the line only once per
			// break: skip the bump for the '\n' that directly follows a '\r'.
			if !(s.char == '\n' && prevCR) {
				s.pos.Line++
			}
			s.pos.Column = 0
			prevCR = s.char == '\r'
		} else {
			prevCR = false
		}
		if err := s.next(); err != nil {
			return err
		}
	}
	return nil
}

//
// scanner method helpers
//

// switch2 is a helper function to evaluate an assignment expression.
// It takes two token.Types and returns a single token.Token.
func (s *Scanner) switch2(tok0, tok1 token.Type) token.Token {
	if s.char == '=' {
		if err := s.next(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}
		return token.New(tok1, tok1.String(), s.start)
	}
	return token.New(tok0, tok0.String(), s.start)
}

// switch3 is a helper function to evaluate an assignment expression.
func (s *Scanner) switch3(tok0, tok1 token.Type, ch2 rune, tok2 token.Type) token.Token {
	if s.char == '=' {
		if err := s.next(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}
		return token.New(tok1, tok1.String(), s.start)
	}
	if s.char == ch2 {
		if err := s.next(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}
		return token.New(tok2, tok2.String(), s.start)
	}
	return token.New(tok0, tok0.String(), s.start)
}

// switch4 is a helper function to evaluate an assignment expression.
func (s *Scanner) switch4(tok0, tok1 token.Type, ch2 rune, tok2, tok3 token.Type) token.Token {
	if s.char == '=' {
		if err := s.next(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}
		return token.New(tok1, tok1.String(), s.start)
	}
	if s.char == ch2 {
		if err := s.next(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}
		if s.char == '=' {
			if err := s.next(); err != nil {
				return token.New(token.ERROR, err.Error(), s.pos)
			}
			return token.New(tok3, tok3.String(), s.start)
		}
		return token.New(tok2, tok2.String(), s.start)
	}
	return token.New(tok0, tok0.String(), s.start)
}

// nextIdentifier scans an identifier or keyword.
func (s *Scanner) nextIdentifier() token.Token {
	buffer := bytes.NewBufferString("")

	for letterOrDigit(s.char) {
		buffer.WriteRune(s.char)
		if err := s.next(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}
	}

	lit := buffer.String()
	typ := token.IDENT
	if len(lit) > 1 { // single-character identifiers are never keywords
		typ = token.Lookup(lit)
	}
	return token.New(typ, lit, s.start)
}

// nextNumber parse a number.
func (s *Scanner) nextNumber(decimal bool) token.Token {
	if decimal {

		buffer := bytes.NewBufferString(".")

		for digit(s.char) {
			buffer.WriteRune(s.char)
			if err := s.next(); err != nil {
				return token.New(token.ERROR, err.Error(), s.pos)
			}
		}
		return token.New(token.FLOAT, buffer.String(), s.start)
	}

	tok := token.INT
	buffer := bytes.NewBufferString("")

	for digit(s.char) || s.char == '.' {
		buffer.WriteRune(s.char)
		if s.char == '.' {
			tok = token.FLOAT
		}
		if err := s.next(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}
	}
	return token.New(tok, buffer.String(), s.start)
}

// nextEOF parse a EOF.
func (s *Scanner) nextEOF() token.Token {
	return token.New(token.EOF, token.EOF.String(), s.start)
}

// nextString parse a string constant.
func (s *Scanner) nextString() token.Token {
	buffer := bytes.NewBufferString("")

	// skip '"'
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}

	for s.char != '"' && !endOfLine(s.char) && s.char != reader.EOF {
		buffer.WriteRune(s.char)
		if err := s.next(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}
	}

	// A string that hits a newline or EOF before its closing quote is
	// unterminated. The EOF guard above is essential: without it the loop spins
	// forever on s.char == reader.EOF (which is neither '"' nor a newline),
	// appending U+FFFD each turn and growing the buffer without bound — a single
	// unterminated literal at end of input would exhaust memory. Scanning runs
	// ahead of the execution step loop, so --timeout/--max-steps cannot stop it.
	if s.char != '"' {
		return token.New(token.ERROR, "string has no closing quote", s.start)
	}
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return token.New(token.STRING, buffer.String(), s.start)
}

// nextColon parse a ':'.
func (s *Scanner) nextColon() token.Token {
	// skip ':'
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return s.switch2(token.COLON, token.DEFINE)
}

// nextPeriod parse a '.'.
func (s *Scanner) nextPeriod() token.Token {
	// skip '.'
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}

	if digit(s.char) {
		return s.nextNumber(true)
	}

	if s.char == '.' {
		if err := s.next(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}
		if s.char == '.' {
			if err := s.next(); err != nil {
				return token.New(token.ERROR, err.Error(), s.pos)
			}
			return token.New(token.ELLIPSIS, "...", s.start)
		}
		return token.New(token.ERROR, "unexpected "+string(s.char), s.pos)
	}
	return token.New(token.PERIOD, token.PERIOD.String(), s.start)
}

// nextComma parse a ','.
func (s *Scanner) nextComma() token.Token {
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return token.New(token.COMMA, token.COMMA.String(), s.start)
}

// nextOpenParen parse a '('.
func (s *Scanner) nextOpenParen() token.Token {
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return token.New(token.LPAREN, token.LPAREN.String(), s.start)
}

// nextCloseParen parse a ')'.
func (s *Scanner) nextCloseParen() token.Token {
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return token.New(token.RPAREN, token.RPAREN.String(), s.start)
}

// nextOpenBracket parse a '['.
func (s *Scanner) nextOpenBracket() token.Token {
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return token.New(token.LBRACK, token.LBRACK.String(), s.start)
}

// nextCloseBracket parse a ']'.
func (s *Scanner) nextCloseBracket() token.Token {
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return token.New(token.RBRACK, token.RBRACK.String(), s.start)
}

// nextOpenBrace parse a '{'.
func (s *Scanner) nextOpenBrace() token.Token {
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return token.New(token.LBRACE, token.LBRACE.String(), s.start)
}

// nextCloseBrace parse a '}'.
func (s *Scanner) nextCloseBrace() token.Token {
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return token.New(token.RBRACE, token.RBRACE.String(), s.start)
}

// nextPlus parse a '+'.
func (s *Scanner) nextPlus() token.Token {
	// skip '+'
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return s.switch3(token.ADD, token.AddAssign, '+', token.INC)
}

// nextDash parse a '-'.
func (s *Scanner) nextDash() token.Token {
	// skip '-'
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return s.switch3(token.SUB, token.SubAssign, '-', token.DEC)
}

// nextStar parse a '*'.
func (s *Scanner) nextStar() token.Token {
	// skip '*'
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return s.switch2(token.MUL, token.MulAssign)
}

// nextSlash parse a '/'.
func (s *Scanner) nextSlash() token.Token {
	// skip '/'
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	if s.char == '/' || s.char == '*' {
		return s.nextComment()
	}
	return s.switch2(token.QUO, token.QuoAssign)
}

// nextComment parse comment styles '//' or '/**/'.
func (s *Scanner) nextComment() token.Token {
	// '/' already consumed; the comment starts at s.start.
	buffer := bytes.NewBufferString("")

	//- style comment
	if s.char == '/' {
		if err := s.next(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}

		if err := s.skipSpaces(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}

		for !endOfLine(s.char) && s.char != reader.EOF {
			buffer.WriteRune(s.char)
			if err := s.next(); err != nil {
				return token.New(token.ERROR, err.Error(), s.pos)
			}
		}
		return token.New(token.COMMENT, buffer.String(), s.start)
	}

	/*- style comment */
	if s.char == '*' {
		if err := s.next(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}

		if err := s.skipSpaces(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}

		ch := s.char

		if err := s.next(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}
		for s.char != reader.EOF {
			if ch == '*' && s.char == '/' {
				// NOTE: the closing '/' is intentionally left unconsumed here to
				// preserve the master run-path behavior byte-for-byte (plan A7):
				// master emits this COMMENT (its span ending just before the '/')
				// followed by a stray QUO '/' token, so an inline block comment
				// like `/* x */ code` is a parse error. That is a latent scanner
				// bug, but fixing it changes `chip run` output, so it is deferred
				// to its own slice rather than smuggled into 3.0b's span work.
				return token.New(token.COMMENT, buffer.String(), s.start)
			}

			buffer.WriteRune(ch)
			ch = s.char

			if err := s.next(); err != nil {
				return token.New(token.ERROR, err.Error(), s.pos)
			}
		}
		return token.New(token.ERROR, "comment never terminated", s.pos)
	}

	return token.New(token.ERROR, "bug: comment parsing", s.pos)
}

// nextPercent parse a '%'.
func (s *Scanner) nextPercent() token.Token {
	// skip '%'
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return s.switch2(token.REM, token.RemAssign)
}

// nextCaret parse a '^'.
func (s *Scanner) nextCaret() token.Token {
	// skip '^'
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return s.switch2(token.XOR, token.XORAssign)
}

// nextLess parse a '<'.
func (s *Scanner) nextLess() token.Token {
	// skip '<'
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	if s.char == '-' {

		// skip '-'
		if err := s.next(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}
		return token.New(token.ARROW, token.ARROW.String(), s.start)
	}
	return s.switch4(token.LSS, token.LEQ, '<', token.SHL, token.ShlAssign)
}

// nextGreater parse a '>'.
func (s *Scanner) nextGreater() token.Token {
	// skip '>'
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return s.switch4(token.GTR, token.GEQ, '>', token.SHR, token.ShrAssign)
}

// nextEqual parse a '='.
func (s *Scanner) nextEqual() token.Token {
	// skip '='
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return s.switch2(token.ASSIGN, token.EQL)
}

// nextBang parse a '!'.
func (s *Scanner) nextBang() token.Token {
	// skip '!'
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return s.switch2(token.NOT, token.NEQ)
}

// nextAmpersand parse a '&'.
func (s *Scanner) nextAmpersand() token.Token {
	// skip '&'
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	if s.char == '^' {
		if err := s.next(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}
		return s.switch2(token.AndNot, token.AndNotAssign)
	}
	return s.switch3(token.AND, token.AndAssign, '&', token.LAND)
}

// nextPipe parse a '|'.
func (s *Scanner) nextPipe() token.Token {
	// skip '|'
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return s.switch3(token.OR, token.ORAssign, '|', token.LOR)
}

func whitespace(ch rune) bool {
	return ch == ' ' || ch == '\t'
}

func endOfLine(ch rune) bool {
	return ch == '\n' || ch == '\r'
}

func letter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch >= 0x80 && unicode.IsLetter(ch)
}

func digit(ch rune) bool {
	return '0' <= ch && ch <= '9' || ch >= 0x80 && unicode.IsDigit(ch)
}

func letterOrDigit(ch rune) bool {
	return letter(ch) || digit(ch)
}
