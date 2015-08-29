package parser

import (
	"fmt"
	"os"

	"github.com/jackspirou/chip/parser/token"
)

// userErr formates an error nicely and writes to the terminal.
func userErr(err error, tok token.Token) {
	fmt.Printf("error: %s on line %d character %d \n", err, tok.Pos.Line, tok.Pos.Column)
	os.Exit(1)
}
