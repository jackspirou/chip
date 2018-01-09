package parser

import (
	"fmt"

	"runtime"
	"strings"
)

// enter writes a message to the terminal displaying the currently executing
// parent function, if tracing is set to true.
func (p *Parser) enter() {
	if p.opts.trace {
		WriteBlanks(p.level)
		fmt.Println("enter " + debug())
		p.level += 4
	}
}

// exit is like enter, but the message says the parent function is exiting.
func (p *Parser) exit() {
	if p.opts.trace {
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
