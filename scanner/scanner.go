//
// SNARL/SCANNER.
//
//		Jack Spirou
//		1 May 2012
//

// SCANNER.  Read characters from source and generate tokens.

package scanner

import (
	"bytes"
	"github.com/jackspirou/chip/reader"
	"github.com/jackspirou/chip/tokens"
	"io"
)

type Scanner struct {
	src  io.Reader
	rdr  *reader.Reader
	ch   rune
	pos  tokens.Position
	chrs chan *reader.Char
	toks chan *tokens.Token
	err  error
}

func NewScanner(src io.Reader) *Scanner {
	s := &Scanner{
		src:  src,
		rdr:  reader.NewReader(src),
		ch:   reader.EOF,               // current Char
		pos:  tokens.NewPosition(1, 0), // default to first line
		chrs: make(chan *reader.Char),
		toks: make(chan *tokens.Token),
		err:  nil, // no errors yet
	}
	s.chrs = s.rdr.GoRead()
	return s
}

func (s *Scanner) GoScan() chan *tokens.Token {
	go s.run()
	return s.toks
}

func (s *Scanner) run() {
	s.next()
	tok, lit := s.scan()
	for tok != tokens.EOF {
		s.toks <- tokens.NewToken(tok, lit, s.pos, s.err)
		if s.err == nil {
			tok, lit = s.scan()
		} else {
			tok = tokens.EOF
		}
	}
	s.toks <- tokens.NewToken(tok, lit, s.pos, s.err)
	close(s.toks)
}

func (s *Scanner) next() rune {
	chr, ok := <-s.chrs
	if !ok {
		panic("scanner.next(): error with channel.")
	}
	if chr.Error() != nil {
		panic("scanner.next(): reader sent scanner this error: " + chr.Error().Error())
	}
	s.ch = chr.Rune()
	s.pos.Column++
	return s.ch
}

func (s *Scanner) skipSpaces() {
	ch := s.ch
	for isWhitespace(ch) || isEndOfLine(ch) {
		if isEndOfLine(ch) {
			s.pos.Line++
			s.pos.Column = 0
		}
		ch = s.next()
	}
	s.ch = ch
}

// Scanner method helpers
func (s *Scanner) switch2(tok0, tok1 tokens.Tokint) (tokens.Tokint, string) {
	if s.ch == '=' {
		s.next()
		return tok1, tok1.String()
	}
	return tok0, tok0.String()
}

func (s *Scanner) switch3(tok0, tok1 tokens.Tokint, ch2 rune, tok2 tokens.Tokint) (tokens.Tokint, string) {
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

func (s *Scanner) switch4(tok0, tok1 tokens.Tokint, ch2 rune, tok2, tok3 tokens.Tokint) (tokens.Tokint, string) {
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

// scan.  Scan the next ch Rune.
func (s *Scanner) scan() (tokens.Tokint, string) {
	s.skipSpaces()
	switch ch := s.ch; {
	case isLetter(ch):
		return s.nextIdentifier()
	case isDigit(ch):
		return s.nextNumber(false)
	default:
		switch ch {
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
			buffer := bytes.NewBufferString("")
			buffer.WriteRune(ch)
			panic("scanner.scan(): The following char is unsupported at this time: " + buffer.String())
		}
	}
	panic("not reached") // for go version 1.0.3 compatibility
}

// Next Identifier.  Set global token to next name.
func (s *Scanner) nextIdentifier() (tokens.Tokint, string) {
	buffer := bytes.NewBufferString("")
	for isLetterOrDigit(s.ch) {
		buffer.WriteRune(s.ch)
		s.next()
	}
	s.next()
	lit := buffer.String()
	if len(lit) > 1 {
		return tokens.Lookup(lit), lit
	}
	return tokens.IDENT, lit
}

// Next Number.
func (s *Scanner) nextNumber(isDecimal bool) (tokens.Tokint, string) {
	if isDecimal {
		buffer := bytes.NewBufferString(".")
		for isDigit(s.ch) {
			buffer.WriteRune(s.ch)
			s.next()
		}
		return tokens.FLOAT, buffer.String()
	}
	tok := tokens.INT
	buffer := bytes.NewBufferString("")
	for isDigit(s.ch) || s.ch == '.' {
		buffer.WriteRune(s.ch)
		if s.ch == '.' {
			tok = tokens.FLOAT
		}
		s.next()
	}
	return tok, buffer.String()
}

func (s *Scanner) nextEOF() (tokens.Tokint, string) {
	return tokens.EOF, "EOF"
}

// Next String Constant. Parse a string constant.
func (s *Scanner) nextString() (tokens.Tokint, string) {
	buffer := bytes.NewBufferString("")
	s.next() // skip '"'
	for s.ch != '"' && !isEndOfLine(s.ch) {
		buffer.WriteRune(s.ch)
		s.next()
	}
	if s.ch != '"' {
		panic("String has no closing quote.")
	}
	s.next()
	return tokens.STRING, buffer.String()
}

// Next Colon. Parses a ':'.
func (s *Scanner) nextColon() (tokens.Tokint, string) {
	s.next() // skip ':'
	return s.switch2(tokens.COLON, tokens.DEFINE)
}

func (s *Scanner) nextPeriod() (tokens.Tokint, string) {
	s.next() // skip '.'
	if isDigit(s.ch) {
		return s.nextNumber(true)
	}
	if s.ch == '.' {
		s.next()
		if s.ch == '.' {
			s.next()
			return tokens.ELLIPSIS, "..."
		} else {
			panic("whats is this?")
		}
	}
	return tokens.PERIOD, "."
}

// Next Comma. Parses a ','.
func (s *Scanner) nextComma() (tokens.Tokint, string) {
	s.next()
	return tokens.COMMA, ","
}

// Next Open Paren. Parse a '('.
func (s *Scanner) nextOpenParen() (tokens.Tokint, string) {
	s.next()
	return tokens.LPAREN, "("
}

// Next Close Paren. Parses a ')'.
func (s *Scanner) nextCloseParen() (tokens.Tokint, string) {
	s.next()
	return tokens.RPAREN, ")"
}

// Next Open Bracket. Parses a '['.
func (s *Scanner) nextOpenBracket() (tokens.Tokint, string) {
	s.next()
	return tokens.LBRACK, "["
}

// Next Close Bracket. Parses a ']'.
func (s *Scanner) nextCloseBracket() (tokens.Tokint, string) {
	s.next()
	return tokens.RBRACK, "]"
}

// Next Open Brace. Parses a '{'.
func (s *Scanner) nextOpenBrace() (tokens.Tokint, string) {
	s.next()
	return tokens.LBRACE, "{"
}

// Next Close Brace. Parses a '}'.
func (s *Scanner) nextCloseBrace() (tokens.Tokint, string) {
	s.next()
	return tokens.RBRACE, "]"
}

// Next Plus. Parse a '+'.
func (s *Scanner) nextPlus() (tokens.Tokint, string) {
	s.next() // skip '+'
	return s.switch3(tokens.ADD, tokens.ADD_ASSIGN, '+', tokens.INC)
}

// Next Dash. Parses a '-'.
func (s *Scanner) nextDash() (tokens.Tokint, string) {
	s.next() // skip '-'
	return s.switch3(tokens.SUB, tokens.SUB_ASSIGN, '-', tokens.DEC)
}

// Next Star. Parse a '*'.
func (s *Scanner) nextStar() (tokens.Tokint, string) {
	s.next() // skip '*'
	return s.switch2(tokens.MUL, tokens.MUL_ASSIGN)
}

// Next Slash. Parse a '/'.
func (s *Scanner) nextSlash() (tokens.Tokint, string) {
	s.next() // skip '/'
	if s.ch == '/' || s.ch == '*' {
		return s.nextComment()
	}
	return s.switch2(tokens.QUO, tokens.QUO_ASSIGN)
}

func (s *Scanner) nextComment() (tokens.Tokint, string) {
	// '/' already consumed
	buffer := bytes.NewBufferString("")
	if s.ch == '/' {
		//- style comment
		s.next()
		for !isEndOfLine(s.ch) && s.ch != reader.EOF {
			buffer.WriteRune(s.ch)
			s.next()
		}
		return tokens.COMMENT, buffer.String()
	}
	if s.ch == '*' {
		/*- style comment */
		s.next()
		ch := s.ch
		s.next()
		for s.ch != reader.EOF {
			if ch == '*' && s.ch == '/' {
				return tokens.COMMENT, buffer.String()
			}
			buffer.WriteRune(ch)
			ch = s.ch
			s.next()
		}
		panic("comment not terminated")
	}
	panic("not reached")
}

func (s *Scanner) nextPercent() (tokens.Tokint, string) {
	s.next()
	return s.switch2(tokens.REM, tokens.REM_ASSIGN)
}

func (s *Scanner) nextCaret() (tokens.Tokint, string) {
	s.next()
	return s.switch2(tokens.XOR, tokens.XOR_ASSIGN)
}

// Next Less. Parse a '<'.
func (s *Scanner) nextLess() (tokens.Tokint, string) {
	s.next()
	if s.ch == '-' {
		s.next()
		return tokens.ARROW, "<-"
	}
	return s.switch4(tokens.LSS, tokens.LEQ, '<', tokens.SHL, tokens.SHL_ASSIGN)
}

// Next Greater. Parse a '>'.
func (s *Scanner) nextGreater() (tokens.Tokint, string) {
	s.next()
	return s.switch4(tokens.GTR, tokens.GEQ, '>', tokens.SHR, tokens.SHR_ASSIGN)
}

// Next Equal. Parse a '='.
func (s *Scanner) nextEqual() (tokens.Tokint, string) {
	s.next()
	return s.switch2(tokens.ASSIGN, tokens.EQL)
}

// Next Bang. Parse a '!'.
func (s *Scanner) nextBang() (tokens.Tokint, string) {
	s.next()
	return s.switch2(tokens.NOT, tokens.NEQ)
}

// Next Ampersand. Parse a '&'.
func (s *Scanner) nextAmpersand() (tokens.Tokint, string) {
	s.next()
	if s.ch == '^' {
		s.next()
		return s.switch2(tokens.AND_NOT, tokens.AND_NOT_ASSIGN)
	}
	return s.switch3(tokens.AND, tokens.AND_ASSIGN, '&', tokens.LAND)
}

// Next Pipe. Parse a '|'.
func (s *Scanner) nextPipe() (tokens.Tokint, string) {
	s.next()
	return s.switch3(tokens.OR, tokens.OR_ASSIGN, '|', tokens.LOR)
}
