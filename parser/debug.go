package parser

import (
	"runtime"
	"strings"
)

// a few string constants to help with printing debugging info
const (
	space  = " "
	enter  = "enter"
	exit   = "exit"
	period = "."
	eol    = "\n"
)

// debug returns the name of the grandparent function by examining the stack at runtime.
func debug() string {
	if pc, _, _, ok := runtime.Caller(2); ok {
		methodPath := runtime.FuncForPC(pc).Name()
		methodPathSlice := strings.Split(methodPath, period)
		return methodPathSlice[len(methodPathSlice)-1]
	}
	return "unknown"
}

// writeBlanks writes a number of blank spaces to the terminal.
func writeBlanks(num int) {
	for num > 0 {
		print(space)
		num--
	}
}

// enter writes a message to the terminal displaying the currently executing
// parent function, if tracing is set to true.
func (p *Parser) enterNext() {
	if p.opts.trace {
		writeBlanks(p.level)
		print(enter + space + debug() + eol)
		p.level += 4
	}
}

// exit is like enter, but the message says the parent function is exiting.
func (p *Parser) exitNext() {
	if p.opts.trace {
		p.level -= 4
		writeBlanks(p.level)
		print(exit + space + debug() + eol)
	}
}
