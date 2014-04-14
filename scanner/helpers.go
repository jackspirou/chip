package scanner

import "unicode"

func isWhitespace(ch rune) bool {
  return ch == ' ' || ch == '\t'
}

func isEndOfLine(ch rune) bool {
  return ch == '\n' || ch == '\r'
}

func isLetter(ch rune) bool {
  return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch >= 0x80 && unicode.IsLetter(ch)
}

func isDigit(ch rune) bool {
  return '0' <= ch && ch <= '9' || ch >= 0x80 && unicode.IsDigit(ch)
}

func isLetterOrDigit(ch rune) bool {
  return isLetter(ch) || isDigit(ch)
}
