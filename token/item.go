package token

// Item represents a token returned from the scanner.
type item struct {
	tok token  // Type, such as itemNumber.
	val string // Value, such as "23.2".
}
