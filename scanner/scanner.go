// Package scanner scans tokens from an UTF-8 io.reader source.
package scanner

import (
	"bytes"
	"fmt"
	"io"
	"unicode"

	"github.com/jackspirou/chip/reader"
	"github.com/jackspirou/chip/token"
)

// Scanner describes a scanner to scan tokens from a UTF-8 io.Reader source.
type Scanner struct {
	read *reader.Reader
	char rune
	pos  token.Pos
}

// New takes an io.Reader and returns a new chip Scanner.
func New(src io.Reader) (*Scanner, error) {
	s := &Scanner{read: reader.New(src), pos: token.NewPos(1, 0)}

	// we must advance the scanner to first position
	if err := s.next(); err != nil {
		return nil, err
	}

	return s, nil
}

// Scan returns the next token.Tokem in the source.
func (s *Scanner) Scan() token.Token {

	// skip any blank spaces
	if err := s.skipSpaces(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}

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
		return token.New(token.ERROR, fmt.Sprintf("unexpected '%c'", s.char), s.pos)
	}
}

// next advances the scanners position and reads the next char.
//
// The only token the next method might return is a token.Error.
func (s *Scanner) next() error {
	char, err := s.read.Read()
	if err != nil && char != reader.EOF {
		return fmt.Errorf("bug: expected EOF got '%c', error: %s", char, err)
	}

	s.char = char
	s.pos.Column++

	return nil
}

// skipSpaces skips all spaces until the next valid character.
func (s *Scanner) skipSpaces() error {
	for whitespace(s.char) || endOfLine(s.char) {
		if endOfLine(s.char) {
			s.pos.Line++
			s.pos.Column = 0
		}
		if err := s.next(); err != nil {
			return err
		}
	}
	return nil
}

// scanner method helpers
//
func (s *Scanner) switch2(tok0, tok1 token.Type) token.Token {
	if s.char == '=' {
		if err := s.next(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}
		return token.New(tok1, tok1.String(), s.pos)
	}
	return token.New(tok0, tok0.String(), s.pos)
}

func (s *Scanner) switch3(tok0, tok1 token.Type, ch2 rune, tok2 token.Type) token.Token {
	if s.char == '=' {
		if err := s.next(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}
		return token.New(tok1, tok1.String(), s.pos)
	}
	if s.char == ch2 {
		if err := s.next(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}
		return token.New(tok2, tok2.String(), s.pos)
	}
	return token.New(tok0, tok0.String(), s.pos)
}

func (s *Scanner) switch4(tok0, tok1 token.Type, ch2 rune, tok2, tok3 token.Type) token.Token {
	if s.char == '=' {
		if err := s.next(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}
		return token.New(tok1, tok1.String(), s.pos)
	}
	if s.char == ch2 {
		if err := s.next(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}
		if s.char == '=' {
			if err := s.next(); err != nil {
				return token.New(token.ERROR, err.Error(), s.pos)
			}
			return token.New(tok3, tok3.String(), s.pos)
		}
		return token.New(tok2, tok2.String(), s.pos)
	}
	return token.New(tok0, tok0.String(), s.pos)
}

// nextIdentifier sets a global token to next name.
func (s *Scanner) nextIdentifier() token.Token {
	pos := s.pos
	buffer := bytes.NewBufferString("")

	for letterOrDigit(s.char) {
		buffer.WriteRune(s.char)
		if err := s.next(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}
	}

	lit := buffer.String()
	if len(lit) > 1 {
		return token.New(token.Lookup(lit), lit, pos)
	}
	return token.New(token.IDENT, lit, s.pos)
}

// nextNumber parse a number.
func (s *Scanner) nextNumber(decimal bool) token.Token {
	pos := s.pos
	if decimal {

		buffer := bytes.NewBufferString(".")

		for digit(s.char) {
			buffer.WriteRune(s.char)
			if err := s.next(); err != nil {
				return token.New(token.ERROR, err.Error(), s.pos)
			}
		}
		return token.New(token.FLOAT, buffer.String(), pos)
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
	return token.New(tok, buffer.String(), pos)
}

// nextEOF parse a EOF.
func (s *Scanner) nextEOF() token.Token {
	return token.New(token.EOF, token.EOF.String(), s.pos)
}

// nextString parse a string constant.
func (s *Scanner) nextString() token.Token {
	pos := s.pos
	buffer := bytes.NewBufferString("")

	// skip '"'
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}

	for s.char != '"' && !endOfLine(s.char) {
		buffer.WriteRune(s.char)
		if err := s.next(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}
	}

	if s.char != '"' {
		return token.New(token.ERROR, "string has no closing quote", pos)
	}
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return token.New(token.STRING, buffer.String(), pos)
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

	pos := s.pos

	if s.char == '.' {
		if err := s.next(); err != nil {
			return token.New(token.ERROR, err.Error(), s.pos)
		}
		if s.char == '.' {
			s.next()
			return token.New(token.ELLIPSIS, "...", pos)
		}
		return token.New(token.ERROR, "unexpected "+string(s.char), s.pos)
	}
	return token.New(token.PERIOD, token.PERIOD.String(), s.pos)
}

// nextComma parse a ','.
func (s *Scanner) nextComma() token.Token {
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return token.New(token.COMMA, token.COMMA.String(), s.pos)
}

// nextOpenParen parse a '('.
func (s *Scanner) nextOpenParen() token.Token {
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return token.New(token.LPAREN, token.LPAREN.String(), s.pos)
}

// nextCloseParen parse a ')'.
func (s *Scanner) nextCloseParen() token.Token {
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return token.New(token.RPAREN, token.RPAREN.String(), s.pos)
}

// nextOpenBracket parse a '['.
func (s *Scanner) nextOpenBracket() token.Token {
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return token.New(token.LBRACK, token.LBRACK.String(), s.pos)
}

// nextCloseBracket parse a ']'.
func (s *Scanner) nextCloseBracket() token.Token {
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return token.New(token.RBRACK, token.RBRACK.String(), s.pos)
}

// nextOpenBrace parse a '{'.
func (s *Scanner) nextOpenBrace() token.Token {
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return token.New(token.LBRACE, token.LBRACE.String(), s.pos)
}

// nextCloseBrace parse a '}'.
func (s *Scanner) nextCloseBrace() token.Token {
	if err := s.next(); err != nil {
		return token.New(token.ERROR, err.Error(), s.pos)
	}
	return token.New(token.RBRACE, token.RBRACE.String(), s.pos)
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
	pos := s.pos
	// '/' already consumed

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
		return token.New(token.COMMENT, buffer.String(), pos)
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
				return token.New(token.COMMENT, buffer.String(), pos)
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
		return token.New(token.ARROW, token.ARROW.String(), s.pos)
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
