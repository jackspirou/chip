package token

import "strconv"

// Type is the set of lexical token types of the chip programming language.
type Type int

// String returns the string corresponding to the token.
// For operators, delimiters, and keywords the string is the actual
// token character sequence (e.g., for the token ADD, the string is
// "+"). For all other tokens the string corresponds to the token
// constant name (e.g. for the token IDENT, the string is "IDENT").
//
func (t Type) String() string {
	s := ""
	if 0 <= t && t < Type(len(tokens)) {
		s = tokens[t]
	}
	if s == "" {
		s = "token(" + strconv.Itoa(int(tok)) + ")"
	}
	return s
}

// A set of constants for precedence-based expression parsing.
// Non-operators have lowest precedence, followed by operators
// starting with precedence 1 up to unary operators. The highest
// precedence serves as "catch-all" precedence for selector,
// indexing, and other operator and delimiter tokens.
//
const (
	LowestPrec  = 0 // non-operators
	UnaryPrec   = 6
	HighestPrec = 7
)

// Precedence returns the token type precedence of the binary
// operator type. If the token type is not a binary operator, the result
// is token.LowestPrec.
//
func (t Type) Precedence() int {
	switch t {
	case LOR:
		return 1
	case LAND:
		return 2
	case EQL, NEQ, LSS, LEQ, GTR, GEQ:
		return 3
	case ADD, SUB, OR, XOR:
		return 4
	case MUL, QUO, REM, SHL, SHR, AND, AndNot:
		return 5
	}
	return LowestPrec
}

// Predicates

// Literal returns true for token types corresponding to identifiers
// and basic type literals; it returns false otherwise.
//
func (t Type) Literal() bool { return literalBegin < t && t < literalEnd }

// Operator returns true for token types corresponding to operators and
// delimiters; it returns false otherwise.
//
func (t Type) Operator() bool { return operatorBegin < t && t < operatorEnd }

// Assignment returns true for token types corresponding to assignments.
func (t Type) Assignment() bool {
	return (assignBegin < t && t < assignEnd) || t == ASSIGN || t == INC || t == DEC
}

// Keyword returns true for token types corresponding to keywords;
// it returns false otherwise.
//
func (t Type) Keyword() bool { return keywordBegin < t && t < keywordEnd }
