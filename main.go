package main

import (
	"bufio"
	"fmt"
	"github.com/jackspirou/chip/helper"
	"github.com/jackspirou/chip/scanner"
	"os"
)

func main() {
	path := "tests/gcd.chp"
	fmt.Println(path)

	// Open a new file and check for errors.
	file, err := os.Open(path)
	helper.Check(err)
	defer file.Close()

	// Set a buffered reader that takes the source file.
	src := bufio.NewReader(file)
	scanr := scanner.NewScanner(src)
	toks := scanr.GoScan()

	for tok := range toks {
		fmt.Println(tok)
	}

	// p := parser.NewParser(src)
	// p.GoParse()
}
