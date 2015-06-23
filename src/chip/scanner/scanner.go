// Package scanner provides a Scanner to read tokens from UTF-8 chip source files.
//

package scanner

import (
	"bytes"
	"io"
	"unicode"

	"github.com/jackspirou/chip/src/chip/reader"
	"github.com/jackspirou/chip/src/chip/token"
)

// Scanner represents a scanner to read tokens from UTF-8 chip source files.
type Scanner struct {

	// input source
	src io.Reader

	// char reader
	rdr *reader.Reader

	// current rune, useful for look-ahead
	ch rune

	// current token position in the source file
	pos token.Pos
}

// NewScanner takes an io.Reader and returns a new Scanner.
func NewScanner(src io.Reader) *Scanner {

	// create a new scanner
	s := &Scanner{

		// set source
		src: src,

		// create a new source reader
		rdr: reader.NewReader(src),

		// set current char to EOF
		ch: reader.EOF,

		// default the position to the first character on the first line
		pos: token.NewPos(1, 0),
	}

	return s
}

// Scan returns the next token and string literal.
func (s *Scanner) Scan() (token.Tokint, string) {

	// skip any blank spaces
	s.skipSpaces()

	// letters yeild identifiers
	if letter(s.ch) {
		return s.nextIdentifier()
	}

	// digits yeild numbers
	if digit(s.ch) {
		return s.nextNumber(false)
	}

	// switch on a rune
	switch s.ch {
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
		return token.ERROR, "unexpected " + string(s.ch)
	}
}

// next advances the scanners position, and reads/returns the next char.
func (s *Scanner) next() error {

	// read the next char
	ch, err := s.rdr.Read()

	// error check
	if err != nil {

		// check that reader's last char was EOF
		if ch != reader.EOF {

			// reader has a bug...
			panic("scanner: the reader's last char should have been EOF, reader has a bug...")
		}

		return err
	}

	// set the reader char equal to the scanner char
	s.ch = ch

	// advance the scanners position
	s.pos.Column++

	return nil
}

func (s *Scanner) skipSpaces() {
	for whitespace(s.ch) || endOfLine(s.ch) {
		if endOfLine(s.ch) {
			s.pos.Line++
			s.pos.Column = 0
		}
		s.next()
	}
}

// Scanner method helpers
func (s *Scanner) switch2(tok0, tok1 token.Tokint) (token.Tokint, string) {
	if s.ch == '=' {
		s.next()
		return tok1, tok1.String()
	}
	return tok0, tok0.String()
}

func (s *Scanner) switch3(tok0, tok1 token.Tokint, ch2 rune, tok2 token.Tokint) (token.Tokint, string) {
	if s.ch == '=' {
		s.next()
		return tok1, tok1.String()
	}
	if s.ch == ch2 {
		s.next()
		return tok2, tok2.String()
	}
	return tok0, tok0.String()
}

func (s *Scanner) switch4(tok0, tok1 token.Tokint, ch2 rune, tok2, tok3 token.Tokint) (token.Tokint, string) {
	if s.ch == '=' {
		s.next()
		return tok1, tok1.String()
	}
	if s.ch == ch2 {
		s.next()
		if s.ch == '=' {
			s.next()
			return tok3, tok3.String()
		}
		return tok2, tok2.String()
	}
	return tok0, tok0.String()
}

// Next Identifier.  Set global token to next name.
func (s *Scanner) nextIdentifier() (token.Tokint, string) {
	buffer := bytes.NewBufferString("")
	for letterOrDigit(s.ch) {
		buffer.WriteRune(s.ch)
		s.next()
	}
	// s.next() - bug? only working because of whitespaces?
	lit := buffer.String()
	if len(lit) > 1 {
		return token.Lookup(lit), lit
	}
	return token.IDENT, lit
}

// Next Number.
func (s *Scanner) nextNumber(decimal bool) (token.Tokint, string) {
	if decimal {
		buffer := bytes.NewBufferString(".")
		for digit(s.ch) {
			buffer.WriteRune(s.ch)
			s.next()
		}
		return token.FLOAT, buffer.String()
	}
	tok := token.INT
	buffer := bytes.NewBufferString("")
	for digit(s.ch) || s.ch == '.' {
		buffer.WriteRune(s.ch)
		if s.ch == '.' {
			tok = token.FLOAT
		}
		s.next()
	}
	return tok, buffer.String()
}

func (s *Scanner) nextEOF() (token.Tokint, string) {
	return token.EOF, token.EOF.String()
}

// Next String Constant. Parse a string constant.
func (s *Scanner) nextString() (token.Tokint, string) {
	buffer := bytes.NewBufferString("")
	s.next() // skip '"'
	for s.ch != '"' && !endOfLine(s.ch) {
		buffer.WriteRune(s.ch)
		s.next()
	}
	if s.ch != '"' {
		return token.ERROR, "String has no closing quote."
	}
	s.next()
	return token.STRING, buffer.String()
}

// Next Colon. Parses a ':'.
func (s *Scanner) nextColon() (token.Tokint, string) {
	s.next() // skip ':'
	return s.switch2(token.COLON, token.DEFINE)
}

func (s *Scanner) nextPeriod() (token.Tokint, string) {
	s.next() // skip '.'
	if digit(s.ch) {
		return s.nextNumber(true)
	}
	if s.ch == '.' {
		s.next()
		if s.ch == '.' {
			s.next()
			return token.ELLIPSIS, "..."
		} else {
			return token.ERROR, "Unexpected " + string(s.ch)
		}
	}
	return token.PERIOD, token.PERIOD.String()
}

// Next Comma. Parses a ','.
func (s *Scanner) nextComma() (token.Tokint, string) {
	s.next()
	return token.COMMA, token.COMMA.String()
}

// Next Open Paren. Parse a '('.
func (s *Scanner) nextOpenParen() (token.Tokint, string) {
	s.next()
	return token.LPAREN, token.LPAREN.String()
}

// Next Close Paren. Parses a ')'.
func (s *Scanner) nextCloseParen() (token.Tokint, string) {
	s.next()
	return token.RPAREN, token.RPAREN.String()
}

// Next Open Bracket. Parses a '['.
func (s *Scanner) nextOpenBracket() (token.Tokint, string) {
	s.next()
	return token.LBRACK, token.LBRACK.String()
}

// Next Close Bracket. Parses a ']'.
func (s *Scanner) nextCloseBracket() (token.Tokint, string) {
	s.next()
	return token.RBRACK, token.RBRACK.String()
}

// Next Open Brace. Parses a '{'.
func (s *Scanner) nextOpenBrace() (token.Tokint, string) {
	s.next()
	return token.LBRACE, token.LBRACE.String()
}

// Next Close Brace. Parses a '}'.
func (s *Scanner) nextCloseBrace() (token.Tokint, string) {
	s.next()
	return token.RBRACE, token.RBRACE.String()
}

// Next Plus. Parse a '+'.
func (s *Scanner) nextPlus() (token.Tokint, string) {
	s.next() // skip '+'
	return s.switch3(token.ADD, token.AddAssign, '+', token.INC)
}

// Next Dash. Parses a '-'.
func (s *Scanner) nextDash() (token.Tokint, string) {
	s.next() // skip '-'
	return s.switch3(token.SUB, token.SubAssign, '-', token.DEC)
}

// Next Star. Parse a '*'.
func (s *Scanner) nextStar() (token.Tokint, string) {
	s.next() // skip '*'
	return s.switch2(token.MUL, token.MulAssign)
}

// Next Slash. Parse a '/'.
func (s *Scanner) nextSlash() (token.Tokint, string) {
	s.next() // skip '/'
	if s.ch == '/' || s.ch == '*' {
		return s.nextComment()
	}
	return s.switch2(token.QUO, token.QuoAssign)
}

func (s *Scanner) nextComment() (token.Tokint, string) {
	// '/' already consumed
	buffer := bytes.NewBufferString("")
	if s.ch == '/' {
		//- style comment
		s.next()
		for !endOfLine(s.ch) && s.ch != reader.EOF {
			buffer.WriteRune(s.ch)
			s.next()
		}
		return token.COMMENT, buffer.String()
	}
	if s.ch == '*' {
		/*- style comment */
		s.next()
		ch := s.ch
		s.next()
		for s.ch != reader.EOF {
			if ch == '*' && s.ch == '/' {
				return token.COMMENT, buffer.String()
			}
			buffer.WriteRune(ch)
			ch = s.ch
			s.next()
		}
		return token.ERROR, "Comment is never terminated"
	}
	panic("not reached")
}

func (s *Scanner) nextPercent() (token.Tokint, string) {
	s.next()
	return s.switch2(token.REM, token.RemAssign)
}

func (s *Scanner) nextCaret() (token.Tokint, string) {
	s.next()
	return s.switch2(token.XOR, token.XORAssign)
}

// Next Less. Parse a '<'.
func (s *Scanner) nextLess() (token.Tokint, string) {
	s.next()
	if s.ch == '-' {
		s.next()
		return token.ARROW, token.ARROW.String()
	}
	return s.switch4(token.LSS, token.LEQ, '<', token.SHL, token.ShlAssign)
}

// Next Greater. Parse a '>'.
func (s *Scanner) nextGreater() (token.Tokint, string) {
	s.next()
	return s.switch4(token.GTR, token.GEQ, '>', token.SHR, token.ShrAssign)
}

// Next Equal. Parse a '='.
func (s *Scanner) nextEqual() (token.Tokint, string) {
	s.next()
	return s.switch2(token.ASSIGN, token.EQL)
}

// Next Bang. Parse a '!'.
func (s *Scanner) nextBang() (token.Tokint, string) {
	s.next()
	return s.switch2(token.NOT, token.NEQ)
}

// Next Ampersand. Parse a '&'.
func (s *Scanner) nextAmpersand() (token.Tokint, string) {
	s.next()
	if s.ch == '^' {
		s.next()
		return s.switch2(token.AndNot, token.AndNotAssign)
	}
	return s.switch3(token.AND, token.AndAssign, '&', token.LAND)
}

// Next Pipe. Parse a '|'.
func (s *Scanner) nextPipe() (token.Tokint, string) {
	s.next()
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
