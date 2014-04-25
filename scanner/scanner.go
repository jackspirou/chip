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
	"github.com/jackspirou/chip/token"
	"io"
)

type Scanner struct {
	src  io.Reader
	rdr  *reader.Reader
	ch   rune
	pos  token.Position
	chrs chan rune
	errs chan error
	toks chan *token.Tok
	err  error
}

func NewScanner(src io.Reader) *Scanner {
	s := &Scanner{
		src:  src,
		rdr:  reader.NewReader(src),
		ch:   reader.EOF,              // current Char
		pos:  token.NewPosition(1, 0), // default to first line
		chrs: make(chan rune),
		errs: make(chan error),
		toks: make(chan *token.Tok),
		err:  nil, // no errors yet
	}
	s.chrs, s.errs = s.rdr.GoRead()
	return s
}

func (s *Scanner) GoScan() chan *token.Tok {
	go s.run()
	return s.toks
}

func (s *Scanner) run() {
	s.next()
	tok, lit := s.scan()
	for tok != token.EOF {
		if tok == token.ERROR {
			if s.err != nil {
				lit = s.err.Error()
			}
			s.toks <- token.NewTok(tok, lit, s.pos)
			tok = token.EOF
			lit = token.EOF.String()
		} else {
			s.toks <- token.NewTok(tok, lit, s.pos)
			tok, lit = s.scan()
		}
	}
	s.toks <- token.NewTok(tok, lit, s.pos)
	close(s.toks)
}

func (s *Scanner) next() rune {
	select {
	case ch := <-s.chrs:
		s.ch = ch
		s.pos.Column++
		return s.ch
	case err := <-s.errs:
		s.err = err
		// flush reader channel for EOF.
		for ch := range s.chrs {
			s.ch = ch
		}
		if s.ch != reader.EOF {
			// Reader's last character wasn't EOF.  Reader has a bug...
			panic("CHIP BUG: The reader package's last character was not EOF.")
		}
		return s.ch
	}
	panic("not reached")
}

func (s *Scanner) skipSpaces() {
	for isWhitespace(s.ch) || isEndOfLine(s.ch) {
		if isEndOfLine(s.ch) {
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

// scan.  Scan the next ch Rune.
func (s *Scanner) scan() (token.Tokint, string) {
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
			return token.ERROR, "Unexpected " + string(ch)
		}
	}
	panic("not reached") // for go version 1.0.3 compatibility
}

// Next Identifier.  Set global token to next name.
func (s *Scanner) nextIdentifier() (token.Tokint, string) {
	buffer := bytes.NewBufferString("")
	for isLetterOrDigit(s.ch) {
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
func (s *Scanner) nextNumber(isDecimal bool) (token.Tokint, string) {
	if isDecimal {
		buffer := bytes.NewBufferString(".")
		for isDigit(s.ch) {
			buffer.WriteRune(s.ch)
			s.next()
		}
		return token.FLOAT, buffer.String()
	}
	tok := token.INT
	buffer := bytes.NewBufferString("")
	for isDigit(s.ch) || s.ch == '.' {
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
	for s.ch != '"' && !isEndOfLine(s.ch) {
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
	if isDigit(s.ch) {
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
	return s.switch3(token.ADD, token.ADD_ASSIGN, '+', token.INC)
}

// Next Dash. Parses a '-'.
func (s *Scanner) nextDash() (token.Tokint, string) {
	s.next() // skip '-'
	return s.switch3(token.SUB, token.SUB_ASSIGN, '-', token.DEC)
}

// Next Star. Parse a '*'.
func (s *Scanner) nextStar() (token.Tokint, string) {
	s.next() // skip '*'
	return s.switch2(token.MUL, token.MUL_ASSIGN)
}

// Next Slash. Parse a '/'.
func (s *Scanner) nextSlash() (token.Tokint, string) {
	s.next() // skip '/'
	if s.ch == '/' || s.ch == '*' {
		return s.nextComment()
	}
	return s.switch2(token.QUO, token.QUO_ASSIGN)
}

func (s *Scanner) nextComment() (token.Tokint, string) {
	// '/' already consumed
	buffer := bytes.NewBufferString("")
	if s.ch == '/' {
		//- style comment
		s.next()
		for !isEndOfLine(s.ch) && s.ch != reader.EOF {
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
	return s.switch2(token.REM, token.REM_ASSIGN)
}

func (s *Scanner) nextCaret() (token.Tokint, string) {
	s.next()
	return s.switch2(token.XOR, token.XOR_ASSIGN)
}

// Next Less. Parse a '<'.
func (s *Scanner) nextLess() (token.Tokint, string) {
	s.next()
	if s.ch == '-' {
		s.next()
		return token.ARROW, token.ARROW.String()
	}
	return s.switch4(token.LSS, token.LEQ, '<', token.SHL, token.SHL_ASSIGN)
}

// Next Greater. Parse a '>'.
func (s *Scanner) nextGreater() (token.Tokint, string) {
	s.next()
	return s.switch4(token.GTR, token.GEQ, '>', token.SHR, token.SHR_ASSIGN)
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
		return s.switch2(token.AND_NOT, token.AND_NOT_ASSIGN)
	}
	return s.switch3(token.AND, token.AND_ASSIGN, '&', token.LAND)
}

// Next Pipe. Parse a '|'.
func (s *Scanner) nextPipe() (token.Tokint, string) {
	s.next()
	return s.switch3(token.OR, token.OR_ASSIGN, '|', token.LOR)
}
