package main

import (
	"bufio"
	"fmt"
	"github.com/JackSpirou/chip/helper"
	"github.com/JackSpirou/chip/parser"
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
	parser := parser.NewParser(src)
	parser.GoParse()
}
