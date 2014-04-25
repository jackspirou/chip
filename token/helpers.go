package token

// Lookup maps an identifier to its keyword token or IDENT (if not a keyword).
func Lookup(ident string) Tokint {
  if tok, is_keyword := keywords[ident]; is_keyword {
    return tok
  }
  return IDENT
}
