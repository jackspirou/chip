package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/jackspirou/chip/src/chip/parser"
)

func main() {
	path := "tests/gcd.chp"
	fmt.Println(path)

	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	src := bufio.NewReader(file)
	parser := parser.NewParser(src)
	parser.GoParse()
}
