package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/jackspirou/chip/parser"
)

func main() {

	path := "test/gcd.chp"
	fmt.Println(path)

	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	p, err := parser.New(bufio.NewReader(file))
	if err != nil {
		panic(err)
	}

	p.Tracing = true

	p.Parse()

}
