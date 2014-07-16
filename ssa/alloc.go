package ssa

import "strconv"

type Allocator struct {
	registers *Register // Linked stack of Register's to be allocated.
	Zero      *Register
	counter   int
}

func NewAllocator() *Allocator {
	a := new(Allocator)
	a.Zero = NewRegister("$zero", nil, true)
	a.counter = 1
	a.registers = NewRegister("$r0", NewRegister("$r1", nil, false), false)

	return a
}

// Request.  Request a Mips register.
func (this *Allocator) Request() *Register {

	if this.registers.next == nil {
		this.counter++
		this.registers.next = NewRegister("$r"+strconv.Itoa(this.counter), nil, false)
	}

	temp := this.registers
	temp.used = true
	this.registers = this.registers.next

	// fmt.Println("Requested " + temp.String() + " : counter at " + strconv.Itoa(this.counter))

	return temp
}

// Release.  Release a Mips register.
func (this *Allocator) Release(reg *Register) {
	if !reg.IsUsed() {
		panic("Empty Stack Exception")
	}
	reg.used = false
	temp := this.registers
	this.registers = reg
	this.registers.next = temp

	// fmt.Println("Released " + temp.String())
}

// Register.  Mips registers.
type Register struct {
	name string    // Printable name of this Register.
	used bool      // Next Register.
	next *Register // Register use state.
}

// Constructor.
func NewRegister(name string, next *Register, used bool) *Register {
	return &Register{name: name, next: next, used: used}
}

// Is Used. Checks if the register is in use
func (this *Register) IsUsed() bool {
	return this.used
}

// To String.  Returns the name of the register.
func (this *Register) String() string {
	return this.name
}
