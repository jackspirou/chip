package main

import (
	"bufio"
	"github.com/jackspirou/chip/scanner"
	"github.com/jackspirou/chip/support"
	"os"
	"fmt"
)

func main() {
	path := "tests/gcd.chp"
	fmt.Println(path)

	// Open a new file and check for errors.
	file, err := os.Open(path)
	support.Check(err)
	defer file.Close()

	// Set a buffered reader that takes the source file.
	src := bufio.NewReader(file)
	s := scanner.NewScanner(src)
	toks := s.GoScan()

	for tok := range toks {
		fmt.Println(tok)
	}


/*
		r := reader.NewReader(src)
		chrs := r.GoRead()

		// Print
		for chr := range chrs {
			if(chr.Error() == nil) {
				fmt.Printf("%c", chr.Rune())
			}
		}
*/
}
