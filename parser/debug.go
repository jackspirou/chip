package parser

import (
	"fmt"
	"github.com/jackspirou/chip/config"
	"github.com/jackspirou/chip/helper"
	//"reflect"
	"runtime"
	"strings"
)

//  ENTER. If we're TRACING, then write a message saying METHOD has started.
func (p *Parser) enter() {
	if config.Tracing {
		helper.WriteBlanks(p.level)
		fmt.Println("Enter " + debug())
		p.level += 4
	}
}

//  EXIT. Like EXIT, but here the message says METHOD has finished.
func (p *Parser) exit() {
	if config.Tracing {
		p.level -= 4
		helper.WriteBlanks(p.level)
		fmt.Println("Exit  " + debug())
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
