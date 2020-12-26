package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/jackspirou/chip/parser"
)

func main() {

	path := "test/simple_add.chp"

	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	p, err := parser.New(bufio.NewReader(file), parser.Trace())
	if err != nil {
		panic(err)
	}

	if err = p.Parse(); err != nil {
		fmt.Println("we are freaking out here!")
		panic(err)
	}

	return
}
