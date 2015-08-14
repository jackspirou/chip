package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/jackspirou/chip/src/chip/scanner"
	"github.com/jackspirou/chip/src/chip/token"
)

func main() {
	path := "test/gcd.chp"
	fmt.Println(path)

	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	scan, err := scanner.New(bufio.NewReader(file))
	if err != nil {
		panic(err)
	}

	for {
		tok, lit := scan.Scan()
		fmt.Println(tok.String() + " " + lit)
		if tok == token.ERROR || tok == token.EOF {
			break
		}
	}
}
