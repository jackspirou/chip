package parser

import (
	"fmt"

	"runtime"
	"strings"
)

const (
	Tracing = true //  Enable or disable ENTER/EXIT.
)

//  ENTER. If we're TRACING, then write a message saying METHOD has started.
func (p *Parser) enter() {
	if Tracing {
		helper.WriteBlanks(p.level)
		fmt.Println("Enter " + debug())
		p.level += 4
	}
}

//  EXIT. Like EXIT, but here the message says METHOD has finished.
func (p *Parser) exit() {
	if Tracing {
		p.level -= 4
		WriteBlanks(p.level)
		fmt.Println("Exit  " + debug())
	}
}

// WriteBlanks writes a number of blank spaces to the terminal.
func WriteBlanks(num int) {
	for num > 0 {
		fmt.Printf(" ")
		num--
	}
}

func debug() string {
	pc, _, _, ok := runtime.Caller(2)
	if ok {
		methodPath := runtime.FuncForPC(pc).Name()
		methodPathSlice := strings.Split(methodPath, ".")
		return methodPathSlice[len(methodPathSlice)-1]
	}
	return "unknown"
}
