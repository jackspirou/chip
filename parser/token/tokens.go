package token

// The list of token types.
const (
	ILLEGAL Type = iota
	ERROR        // error
	EOF          // end of file
	COMMENT      // "//" or "/* */"

	// identifiers and basic type literals, they stand for classes of literals
	literalBegin
	IDENT  // main
	INT    // 12345
	FLOAT  // 123.45
	IMAG   // 123.45i
	CHAR   // 'a'
	STRING // "abc"
	literalEnd

	// operators and delimiters
	operatorBegin
	ADD // +
	SUB // -
	MUL // *
	QUO // /
	REM // %

	AND    // &
	OR     // |
	XOR    // ^
	SHL    // <<
	SHR    // >>
	AndNot // &^

	assignBegin
	AddAssign // +=
	SubAssign // -=
	MulAssign // *=
	QuoAssign // /=
	RemAssign // %=

	AndAssign    // &=
	ORAssign     // |=
	XORAssign    // ^=
	ShlAssign    // <<=
	ShrAssign    // >>=
	AndNotAssign // &^=
	assignEnd

	LAND // &&
	LOR  // ||

	comparisonBegin
	EQL    // ==
	LSS    // <
	GTR    // >
	ASSIGN // =
	NOT    // !
	NEQ    // !=
	LEQ    // <=
	GEQ    // >=
	comparisonEnd

	DEFINE   // :=
	ELLIPSIS // ...
	ARROW    // <-
	INC      // ++
	DEC      // --

	LPAREN // (
	LBRACK // [
	LBRACE // {
	COMMA  // ,
	PERIOD // .

	RPAREN    // )
	RBRACK    // ]
	RBRACE    // }
	SEMICOLON // ;
	COLON     // :
	operatorEnd

	// keywords
	keywordBegin
	BREAK    // break
	CASE     // case
	CHAN     // chan
	CONST    // const
	CONTINUE // continue

	DEFAULT     // default
	DEFER       // defer
	ELSE        // else
	FALLTHROUGH // fallthrough
	FOR         // for

	FUNC   // func
	GO     // go
	GOTO   // goto
	IF     // if
	IMPORT // import

	INTERFACE // interfact
	IOTA      // iota
	MAP       // map
	PACKAGE   // package
	RANGE     // range
	RETURN    // return

	SELECT // select
	STRUCT // struct
	SWITCH // switch
	TYPE   // type
	VAR    // var
	keywordEnd
)

var tokens = [...]string{
	ILLEGAL: "ILLEGAL",
	ERROR:   "ERROR",
	EOF:     "EOF",
	COMMENT: "COMMENT",

	IDENT:  "IDENT",
	INT:    "INT",
	FLOAT:  "FLOAT",
	IMAG:   "IMAG",
	CHAR:   "CHAR",
	STRING: "STRING",

	ADD: "+",
	SUB: "-",
	MUL: "*",
	QUO: "/",
	REM: "%",

	AND:    "&",
	OR:     "|",
	XOR:    "^",
	SHL:    "<<",
	SHR:    ">>",
	AndNot: "&^",

	AddAssign: "+=",
	SubAssign: "-=",
	MulAssign: "*=",
	QuoAssign: "/=",
	RemAssign: "%=",

	AndAssign:    "&=",
	ORAssign:     "|=",
	XORAssign:    "^=",
	ShlAssign:    "<<=",
	ShrAssign:    ">>=",
	AndNotAssign: "&^=",

	LAND:  "&&",
	LOR:   "||",
	ARROW: "<-",
	INC:   "++",
	DEC:   "--",

	EQL:    "==",
	LSS:    "<",
	GTR:    ">",
	ASSIGN: "=",
	NOT:    "!",

	NEQ:      "!=",
	LEQ:      "<=",
	GEQ:      ">=",
	DEFINE:   ":=",
	ELLIPSIS: "...",

	LPAREN: "(",
	LBRACK: "[",
	LBRACE: "{",
	COMMA:  ",",
	PERIOD: ".",

	RPAREN:    ")",
	RBRACK:    "]",
	RBRACE:    "}",
	SEMICOLON: ";",
	COLON:     ":",

	BREAK:    "break",
	CASE:     "case",
	CHAN:     "chan",
	CONST:    "const",
	CONTINUE: "continue",

	DEFAULT:     "default",
	DEFER:       "defer",
	ELSE:        "else",
	FALLTHROUGH: "fallthrough",
	FOR:         "for",

	FUNC:   "func",
	GO:     "go",
	GOTO:   "goto",
	IF:     "if",
	IMPORT: "import",

	INTERFACE: "interface",
	IOTA:      "itoa",
	MAP:       "map",
	PACKAGE:   "package",
	RANGE:     "range",
	RETURN:    "return",

	SELECT: "select",
	STRUCT: "struct",
	SWITCH: "switch",
	TYPE:   "type",
	VAR:    "var",
}

var keywords map[string]Type

func init() {
	keywords = make(map[string]Type)
	for i := keywordBegin + 1; i < keywordEnd; i++ {
		keywords[tokens[i]] = i
	}
}
