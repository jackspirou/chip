package main

import (
	"bufio"
	"os"

	"github.com/jackspirou/chip/parser"
)

func main() {

	path := "test/gcd.chp"

	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	p, err := parser.New(bufio.NewReader(file), parser.Trace())
	if err != nil {
		panic(err)
	}

	if err = p.Execute(); err != nil {
		panic(err)
	}

}
