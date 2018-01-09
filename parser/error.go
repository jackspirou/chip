package parser

import (
	"fmt"
	"os"

	"github.com/jackspirou/chip/token"
)

// userErr formates an error nicely and writes to the terminal.
func userErr(err error, tok token.Token) {
	fmt.Printf("error: %s on line %d character %d \n", err, tok.Line(), tok.Column())
	os.Exit(1)
}
