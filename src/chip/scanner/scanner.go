// Package scanner provides an implimention to scan chip tokens in a UTF-8 source.
//
package scanner

import (
	"bytes"
	"fmt"
	"io"
	"unicode"

	"github.com/jackspirou/chip/src/chip/reader"
	"github.com/jackspirou/chip/src/chip/token"
)

// Scanner represents a scanner to scan chip tokens in a UTF-8 source.
type Scanner struct {
	src  io.Reader
	rdr  *reader.Reader
	char rune // current rune, useful for look-ahead
	pos  token.Pos
}

// New takes an io.Reader and returns a new chip Scanner.
func New(src io.Reader) *Scanner {
	s := &Scanner{src, reader.New(src), reader.EOF, token.NewPos(1, 0)}
	s.next()
	return s
}

// Scan returns the next token.Type and string literal in the source.
func (s *Scanner) Scan() (token.Type, string) {

	// skip any blank spaces
	s.skipSpaces()

	// letters yeild identifiers
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
		return token.ERROR, fmt.Sprintf("unexpected '%c'", s.char)
	}
}

// next advances the scanners position and reads the next char.
func (s *Scanner) next() error {

	char, err := s.rdr.Read()
	if err != nil {
		if char != reader.EOF {
			return fmt.Errorf(
				"%s: expected EOF got '%c', please report this bug", err, char)
		}
		return err
	}

	s.char = char

	// advance position
	s.pos.Column++

	return nil
}

// skipSpaces skips all spaces until the next valid character.
func (s *Scanner) skipSpaces() {
	for whitespace(s.char) || endOfLine(s.char) {
		if endOfLine(s.char) {
			s.pos.Line++
			s.pos.Column = 0
		}
		s.next()
	}
}

// scanner method helpers
//
func (s *Scanner) switch2(tok0, tok1 token.Type) (token.Type, string) {
	if s.char == '=' {
		s.next()
		return tok1, tok1.String()
	}
	return tok0, tok0.String()
}

func (s *Scanner) switch3(tok0, tok1 token.Type, ch2 rune, tok2 token.Type) (token.Type, string) {
	if s.char == '=' {
		s.next()
		return tok1, tok1.String()
	}
	if s.char == ch2 {
		s.next()
		return tok2, tok2.String()
	}
	return tok0, tok0.String()
}

func (s *Scanner) switch4(tok0, tok1 token.Type, ch2 rune, tok2, tok3 token.Type) (token.Type, string) {
	if s.char == '=' {
		s.next()
		return tok1, tok1.String()
	}
	if s.char == ch2 {
		s.next()
		if s.char == '=' {
			s.next()
			return tok3, tok3.String()
		}
		return tok2, tok2.String()
	}
	return tok0, tok0.String()
}

// nextIdentifier sets a global token to next name.
func (s *Scanner) nextIdentifier() (token.Type, string) {
	buffer := bytes.NewBufferString("")
	for letterOrDigit(s.char) {
		buffer.WriteRune(s.char)
		s.next()
	}
	// s.next() - bug? only working because of whitespaces?
	lit := buffer.String()
	if len(lit) > 1 {
		return token.Lookup(lit), lit
	}
	return token.IDENT, lit
}

// nextNumber parse a number.
func (s *Scanner) nextNumber(decimal bool) (token.Type, string) {
	if decimal {
		buffer := bytes.NewBufferString(".")
		for digit(s.char) {
			buffer.WriteRune(s.char)
			s.next()
		}
		return token.FLOAT, buffer.String()
	}
	tok := token.INT
	buffer := bytes.NewBufferString("")
	for digit(s.char) || s.char == '.' {
		buffer.WriteRune(s.char)
		if s.char == '.' {
			tok = token.FLOAT
		}
		s.next()
	}
	return tok, buffer.String()
}

// nextEOF parse a EOF.
func (s *Scanner) nextEOF() (token.Type, string) {
	return token.EOF, token.EOF.String()
}

// nextString parse a string constant.
func (s *Scanner) nextString() (token.Type, string) {
	buffer := bytes.NewBufferString("")
	s.next() // skip '"'
	for s.char != '"' && !endOfLine(s.char) {
		buffer.WriteRune(s.char)
		s.next()
	}
	if s.char != '"' {
		return token.ERROR, "String has no closing quote."
	}
	s.next()
	return token.STRING, buffer.String()
}

// nextColon parse a ':'.
func (s *Scanner) nextColon() (token.Type, string) {
	s.next() // skip ':'
	return s.switch2(token.COLON, token.DEFINE)
}

// nextPeriod parse a '.'.
func (s *Scanner) nextPeriod() (token.Type, string) {
	s.next() // skip '.'
	if digit(s.char) {
		return s.nextNumber(true)
	}
	if s.char == '.' {
		s.next()
		if s.char == '.' {
			s.next()
			return token.ELLIPSIS, "..."
		}
		return token.ERROR, "Unexpected " + string(s.char)
	}
	return token.PERIOD, token.PERIOD.String()
}

// nextComma parse a ','.
func (s *Scanner) nextComma() (token.Type, string) {
	s.next()
	return token.COMMA, token.COMMA.String()
}

// nextOpenParen parse a '('.
func (s *Scanner) nextOpenParen() (token.Type, string) {
	s.next()
	return token.LPAREN, token.LPAREN.String()
}

// nextCloseParen parse a ')'.
func (s *Scanner) nextCloseParen() (token.Type, string) {
	s.next()
	return token.RPAREN, token.RPAREN.String()
}

// nextOpenBracket parse a '['.
func (s *Scanner) nextOpenBracket() (token.Type, string) {
	s.next()
	return token.LBRACK, token.LBRACK.String()
}

// nextCloseBracket parse a ']'.
func (s *Scanner) nextCloseBracket() (token.Type, string) {
	s.next()
	return token.RBRACK, token.RBRACK.String()
}

// nextOpenBrace parse a '{'.
func (s *Scanner) nextOpenBrace() (token.Type, string) {
	s.next()
	return token.LBRACE, token.LBRACE.String()
}

// nextCloseBrace parse a '}'.
func (s *Scanner) nextCloseBrace() (token.Type, string) {
	s.next()
	return token.RBRACE, token.RBRACE.String()
}

// nextPlus parse a '+'.
func (s *Scanner) nextPlus() (token.Type, string) {
	s.next() // skip '+'
	return s.switch3(token.ADD, token.AddAssign, '+', token.INC)
}

// nextDash parse a '-'.
func (s *Scanner) nextDash() (token.Type, string) {
	s.next() // skip '-'
	return s.switch3(token.SUB, token.SubAssign, '-', token.DEC)
}

// nextStar parse a '*'.
func (s *Scanner) nextStar() (token.Type, string) {
	s.next() // skip '*'
	return s.switch2(token.MUL, token.MulAssign)
}

// nextSlash parse a '/'.
func (s *Scanner) nextSlash() (token.Type, string) {
	s.next() // skip '/'
	if s.char == '/' || s.char == '*' {
		return s.nextComment()
	}
	return s.switch2(token.QUO, token.QuoAssign)
}

// nextComment parse comment styles '//' or '/**/'.
func (s *Scanner) nextComment() (token.Type, string) {

	// '/' already consumed

	buffer := bytes.NewBufferString("")

	//- style comment
	if s.char == '/' {
		s.next()
		for !endOfLine(s.char) && s.char != reader.EOF {
			buffer.WriteRune(s.char)
			s.next()
		}
		return token.COMMENT, buffer.String()
	}

	/*- style comment */
	if s.char == '*' {
		s.next()
		ch := s.char
		s.next()
		for s.char != reader.EOF {
			if ch == '*' && s.char == '/' {
				return token.COMMENT, buffer.String()
			}
			buffer.WriteRune(ch)
			ch = s.char
			s.next()
		}
		return token.ERROR, "comment never terminates"
	}

	return token.ERROR, "comment parsing bug, please report this"
}

// nextPercent parse a '%'.
func (s *Scanner) nextPercent() (token.Type, string) {
	s.next() // skip '%'
	return s.switch2(token.REM, token.RemAssign)
}

// nextCaret parse a '^'.
func (s *Scanner) nextCaret() (token.Type, string) {
	s.next() // skip '^'
	return s.switch2(token.XOR, token.XORAssign)
}

// nextLess parse a '<'.
func (s *Scanner) nextLess() (token.Type, string) {
	s.next() // skip '<'
	if s.char == '-' {
		s.next() // skip '-'
		return token.ARROW, token.ARROW.String()
	}
	return s.switch4(token.LSS, token.LEQ, '<', token.SHL, token.ShlAssign)
}

// nextGreater parse a '>'.
func (s *Scanner) nextGreater() (token.Type, string) {
	s.next() // skip '>'
	return s.switch4(token.GTR, token.GEQ, '>', token.SHR, token.ShrAssign)
}

// nextEqual parse a '='.
func (s *Scanner) nextEqual() (token.Type, string) {
	s.next() // skip '='
	return s.switch2(token.ASSIGN, token.EQL)
}

// nextBang parse a '!'.
func (s *Scanner) nextBang() (token.Type, string) {
	s.next() // skip '!'
	return s.switch2(token.NOT, token.NEQ)
}

// nextAmpersand parse a '&'.
func (s *Scanner) nextAmpersand() (token.Type, string) {
	s.next() // skip '&'
	if s.char == '^' {
		s.next() // skip '^'
		return s.switch2(token.AndNot, token.AndNotAssign)
	}
	return s.switch3(token.AND, token.AndAssign, '&', token.LAND)
}

// nextPipe parse a '|'.
func (s *Scanner) nextPipe() (token.Type, string) {
	s.next() // skip '|'
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
