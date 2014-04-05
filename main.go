package main

import (
	"bufio"
	"github.com/jackspirou/chip/reader"
	"github.com/jackspirou/chip/support"
	"fmt"
	"os"
	)

	func main() {
		path := "tests/syntax.chp"

		// Open a new file and check for errors.
		file, err := os.Open(path)
		support.Check(err)
		defer file.Close()

		fmt.Println("Printing File: " + file.Name())
		fmt.Println("")

		// Set a buffered reader that takes the source file.
		src := bufio.NewReader(file)
		r := reader.NewReader(src)
		chrs := r.GoRead()

		// Print
		for chr := range chrs {
			if(chr.Error() == nil) {
				fmt.Printf("%c", chr.Rune())
			}
		}
	}
