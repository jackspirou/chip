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
	"chp/reader"
	"chp/token"
	"strconv"
	"unicode"
)

type Scanner struct {
	reservedNames map[string]int // A hash table.
	token         int            // Integer value for the current token.
	tokenInteger  int            // Current constant integer token value.
	Source        *Source        // Instance of the class Source.
	tokenString   string         // Current constant string token value.
}

func NewScanner(source *Source) *Scanner {
	scanner := new(Scanner)

	scanner.Source = source
	scanner.tokenInteger = 0
	scanner.tokenString = ""
	scanner.reservedNames = make(map[string]int)

	// Place reserved names into hash table map
	scanner.reservedNames["and"] = common.BoldAndToken
	scanner.reservedNames["begin"] = common.BoldBeginToken
	scanner.reservedNames["code"] = common.BoldCodeToken
	scanner.reservedNames["do"] = common.BoldDoToken
	scanner.reservedNames["else"] = common.BoldElseToken
	scanner.reservedNames["end"] = common.BoldEndToken
	scanner.reservedNames["if"] = common.BoldIfToken
	scanner.reservedNames["int"] = common.BoldIntToken
	scanner.reservedNames["or"] = common.BoldOrToken
	scanner.reservedNames["not"] = common.BoldNotToken
	scanner.reservedNames["func"] = common.BoldFuncToken
	scanner.reservedNames["string"] = common.BoldStringToken
	scanner.reservedNames["then"] = common.BoldThenToken
	scanner.reservedNames["value"] = common.BoldValueToken
	scanner.reservedNames["while"] = common.BoldWhileToken
	// scanner.reservedNames["=="] = common.EqualEqualToken

	// Advance the scanner to the first token.
	scanner.NextToken()

	return scanner
}

func (this *Scanner) GetPath() string {
	return this.Source.GetPath()
}

// Get Integer.  Return value of current token integer.
func (this *Scanner) GetInteger() int {
	return this.tokenInteger
}

// Get String.  Return value of current string token.
func (this *Scanner) GetString() string {
	return this.tokenString
}

// Get Token.  Return current value of token.
func (this *Scanner) GetToken() int {
	return this.token
}

// Next Single. Stores token argument.
func (this *Scanner) NextSingle(token int) {
	this.token = token
	this.Source.NextChar()
}

// Is Letter. Checks if argument is a letter.
func (this *Scanner) isLetter(ch rune) bool {
	return unicode.IsLetter(ch)
	// return 'a' <=ch && ch <= 'z' || 'A' <= ch && ch <= 'Z'
}

// Is Digit.  Check if the argument is a digit.
func (this *Scanner) isDigit(ch rune) bool {
	return unicode.IsDigit(ch)
	// return '0' <= ch && ch <= '9'
}

// Is Letter Or Digit.  Check if the current character is a letter or digit.
func (this *Scanner) isLetterOrDigit(ch rune) bool {
	return this.isLetter(ch) || this.isDigit(ch)
}

// Next Token. Gets the next token.
func (this *Scanner) NextToken() {
	this.token = common.IgnoredToken
	for this.token == common.IgnoredToken {
		if this.isLetter(this.Source.GetChar()) {
			this.nextName()
		} else if this.isDigit(this.Source.GetChar()) {
			this.nextIntConstant()
		} else {
			switch this.Source.GetChar() {
			case common.EOFChar:
				this.nextEndFile()
			// Skip spaces and tabs
			case ' ', '	':
				this.Source.NextChar()
			case '"':
				this.nextStringConstant()
			case '#':
				this.nextLineComment()
			case '(':
				this.nextOpenParen()
			case ')':
				this.nextCloseParen()
			case '*':
				this.nextStar()
			case '+':
				this.nextPlus()
			case ',':
				this.nextComma()
			case '-':
				this.nextDash()
			case '/':
				this.nextSlash()
			case ':':
				this.nextColon()
			case ';':
				this.nextSemicolon()
			case '<':
				this.nextLess()
			case '=':
				this.nextEqual()
			case '>':
				this.nextGreater()
			case '[':
				this.nextOpenBracket()
			case ']':
				this.nextCloseBracket()
			case '{':
				this.nextOpenCurlyBracket()
			case '}':
				this.nextCloseCurlyBracket()
			default:
				this.nextIllegal()
			}
		}
	}
}

// Next Close Bracket. Parses a ']'.
func (this *Scanner) nextCloseBracket() {
	this.token = common.CloseBracketToken
	this.Source.NextChar()
}

// Next Close Curly Bracket. Parses a '}'.
func (this *Scanner) nextCloseCurlyBracket() {
	this.token = common.CloseCurlyBracketToken
	this.Source.NextChar()
}

// Next Close Paren. Parses a ')'.
func (this *Scanner) nextCloseParen() {
	this.token = common.CloseParenToken
	this.Source.NextChar()
}

// Next Colon. Parses a ':'.
func (this *Scanner) nextColon() {
	this.token = common.ColonToken
	this.Source.NextChar()
	if this.Source.GetChar() == '=' {
		this.token = common.ColonEqualToken
		this.Source.NextChar()
	}
}

// Next Comma. Parses a ','.
func (this *Scanner) nextComma() {
	this.token = common.CommaToken
	this.Source.NextChar()
}

// Next Line Comment. Parses a comment to the next line.
func (this *Scanner) nextLineComment() {
	for !this.Source.AtLineEnd() {
		this.Source.NextChar()
	}
	this.Source.NextChar()
	this.NextToken()
}

// Next Dash. Parses a '-'.
func (this *Scanner) nextDash() {
	this.token = common.DashToken
	this.Source.NextChar()
}

// Next End File. Parse end.
func (this *Scanner) nextEndFile() {
	this.token = common.EndFileToken
}

// Next Equal. Parse a '='.
func (this *Scanner) nextEqual() {
	this.token = common.EqualToken
	this.Source.NextChar()
	if this.Source.GetChar() == '=' {
		this.token = common.EqualEqualToken
		this.Source.NextChar()
	}
}

// Next Greater. Parse a '>'.
func (this *Scanner) nextGreater() {
	this.token = common.GreaterToken
	this.Source.NextChar()
	if this.Source.GetChar() == '=' {
		this.token = common.GreaterEqualToken
		this.Source.NextChar()
	}
}

// Next Illegal. Parse illegal token.
func (this *Scanner) nextIllegal() {
	this.Source.Error("Illegal token.")
}

// Next Int Constant. Parse an integer constant.
func (this *Scanner) nextIntConstant() {

	buffer := bytes.NewBufferString("")

	for this.isDigit(this.Source.GetChar()) {
		_, err := buffer.WriteRune(this.Source.GetChar())
		common.ErrorCheck(err, "A buffer error occured when parsing an integer constant.")
		this.Source.NextChar()
	}

	this.tokenString = buffer.String()
	parsedInt, err := strconv.Atoi(this.tokenString)
	common.ErrorCheck(err, "Illegal number.")

	this.tokenInteger = parsedInt
	this.token = common.IntConstantToken
}

// Next Less. Parse a '<'.
func (this *Scanner) nextLess() {
	this.token = common.LessToken
	this.Source.NextChar()
	if this.Source.GetChar() == '=' {
		this.token = common.LessEqualToken
		this.Source.NextChar()
	} else if this.Source.GetChar() == '>' {
		this.token = common.LessGreaterToken
		this.Source.NextChar()
	}
}

// Next Name.  Set global token to next name.
func (this *Scanner) nextName() {

	buffer := bytes.NewBufferString("")

	for this.isLetterOrDigit(this.Source.GetChar()) {
		buffer.WriteRune(this.Source.GetChar())
		this.Source.NextChar()
	}

	this.tokenString = buffer.String()

	if _, present := this.reservedNames[this.tokenString]; present {
		this.token = this.reservedNames[this.tokenString]
	} else {
		this.token = common.NameToken
	}

}

// Next Open Bracket. Parse a '['.
func (this *Scanner) nextOpenBracket() {
	this.token = common.OpenBracketToken
	this.Source.NextChar()
}

// Next Open Curly Bracket. Parse a '{'.
func (this *Scanner) nextOpenCurlyBracket() {
	this.token = common.OpenCurlyBracketToken
	this.Source.NextChar()
}

// Next Open Paren. Parse a '('.
func (this *Scanner) nextOpenParen() {
	this.token = common.OpenParenToken
	this.Source.NextChar()
}

// Next Plus. Parse a '+'.
func (this *Scanner) nextPlus() {
	this.token = common.PlusToken
	this.Source.NextChar()
}

// Next Semicolon. Parse a ';'.
func (this *Scanner) nextSemicolon() {
	this.token = common.SemicolonToken
	this.Source.NextChar()
}

// Next Slash. Parse a '/'.
func (this *Scanner) nextSlash() {
	this.token = common.SlashToken
	this.Source.NextChar()
	if this.Source.GetChar() == '/' {
		this.nextLineComment()
	} else if this.Source.GetChar() == '*' {
		this.Source.NextChar()
		for this.token != common.CommentToken {
			if this.Source.GetChar() == '*' {
				this.Source.NextChar()
				if this.Source.GetChar() == '/' {
					this.token = common.CommentToken
				}
			} else if this.Source.AtEndOfFile() {
				this.Source.Error("Unexpected End Of File.")
			} else {
				this.Source.NextChar()
			}
		}
		this.Source.NextChar()
		this.NextToken()
	}
}

// Next Star. Parse a '*'.
func (this *Scanner) nextStar() {
	this.token = common.StarToken
	this.Source.NextChar()
}

// Next String Constant. Parse a string constant.
func (this *Scanner) nextStringConstant() {
	this.token = common.StringConstantToken
	buffer := bytes.NewBufferString("")
	this.Source.NextChar()

	for this.Source.GetChar() != '"' && !this.Source.AtLineEnd() {
		buffer.WriteRune(this.Source.GetChar())
		this.Source.NextChar()
	}

	if this.Source.GetChar() == '"' {
		this.Source.NextChar()
	} else {
		this.Source.Error("String has no closing quote.")
	}
	this.tokenString = buffer.String()
}
