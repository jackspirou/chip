package parser

import (
	"fmt"
	"os"

	"github.com/jackspirou/chip/token"

	"runtime"
	"strings"
)

// enter writes a message to the terminal displaying the currently executing
// parent function, if tracing is set to true.
func (p *Parser) enter() {
	if p.Tracing {
		WriteBlanks(p.level)
		fmt.Println("enter " + debug())
		p.level += 4
	}
}

// exit is like enter, but the message says the parent function is exiting.
func (p *Parser) exit() {
	if p.Tracing {
		p.level -= 4
		WriteBlanks(p.level)
		fmt.Println("exit  " + debug())
	}
}

// WriteBlanks writes a number of blank spaces to the terminal.
func WriteBlanks(num int) {
	for num > 0 {
		fmt.Printf(" ")
		num--
	}
}

// debug returns the name of the grandparent function by examining the stack at
// runtime.
func debug() string {
	pc, _, _, ok := runtime.Caller(2)
	if ok {
		methodPath := runtime.FuncForPC(pc).Name()
		methodPathSlice := strings.Split(methodPath, ".")
		return methodPathSlice[len(methodPathSlice)-1]
	}
	return "unknown"
}

// userErr formates an error nicely and writes to the terminal.
func userErr(err error, tok token.Token) {
	fmt.Printf("error: %s on line %d character %d \n", err, tok.Pos.Line, tok.Pos.Column)
	os.Exit(1)
}
