package ssa

import (
	"log"
	"strconv"
)

// Allocator describes an allocator for registers.
type Allocator struct {
	registers *Register // Linked stack of Register's to be allocated.
	Zero      *Register
	counter   int
}

// NewAllocator returns a new Allocator object.
func NewAllocator() *Allocator {
	return &Allocator{
		registers: NewRegister("$r0", NewRegister("$r1", nil, false), false),
		Zero:      NewRegister("$zero", nil, true),
		counter:   1,
	}
}

// Request requests a register.
func (a *Allocator) Request() *Register {

	if a.registers.next == nil {
		a.counter++
		a.registers.next = NewRegister("$r"+strconv.Itoa(a.counter), nil, false)
	}

	temp := a.registers
	temp.used = true
	a.registers = a.registers.next

	// // p.enter()("Requested " + temp.String() + " : counter at " + strconv.Itoa(a.counter))

	return temp
}

// Release releaes a register.
func (a *Allocator) Release(reg *Register) {
	if !reg.Used() {
		log.Fatal("empty stack exception")
	}
	reg.used = false
	temp := a.registers
	a.registers = reg
	a.registers.next = temp

	// fmt.Println("Released " + temp.String())
}

// Register describes a register. Register is modeled off MIPS registers.
type Register struct {
	name string    // Printable name of this Register.
	used bool      // Next Register.
	next *Register // Register use state.
}

// NewRegister returns a new Register object.
func NewRegister(name string, next *Register, used bool) *Register {
	return &Register{name: name, next: next, used: used}
}

// Used returns true of the register is currently in use.
func (r *Register) Used() bool {
	return r.used
}

// String satisfies the fmt.Stringer interface.
func (r Register) String() string {
	return r.name
}
