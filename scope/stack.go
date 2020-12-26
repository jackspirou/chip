package scope

import (
	"errors"
	"fmt"
)

// stack describes a SymbolTable stack.
type stack struct {
	list []SymbolTable
}

// newStack returns a new stack.
func newStack() *stack {
	return &stack{}
}

// push pushes a new SymbolTable on the stack.
func (s *stack) push(st SymbolTable) {
	s.list = append(s.list, st)
}

// pop pops a SymbolTable off the stack.
func (s *stack) pop() (SymbolTable, error) {
	if s.empty() {
		return SymbolTable{}, errors.New("cannot pop an empty stack")
	}
	top := s.list[len(s.list)-1]
	s.list = s.list[0 : len(s.list)-1]
	return top, nil
}

func (s stack) bottom() (*SymbolTable, error) {
	if s.empty() {
		return nil, errors.New("cannot get the bottom of an empty stack")
	}
	bottom := s.list[0]
	return &bottom, nil
}

func (s stack) peek() (SymbolTable, error) {
	if s.empty() {
		return SymbolTable{}, errors.New("cannot peek on an empty stack")
	}
	return s.list[len(s.list)-1], nil
}

// empty returns true of the stack is empty.
func (s stack) empty() bool {
	return len(s.list) < 1
}

// size returns the number of SymbolTables in the stack.
func (s stack) size() int {
	return len(s.list)
}

// String satisfies the fmt.Stringer interface.
func (s stack) String() string {
	str := ""
	if !s.empty() {
		for _, v := range s.list {
			str += fmt.Sprintf("%v ", v)
		}
		str = str[:len(str)-2]
	}
	return fmt.Sprintf("[%s]", str)
}
