package typ

import "github.com/JackSpirou/chip/token"

// Procedure Type
type ProcedureType struct {
	arity int        //  Number of parameters for this procedure.
	first *Parameter //  Head node of the parameter list.
	last  *Parameter //  Last node of the parameter list.
	value Typ        //  The type this procedure returns.
}

//  Constructor. Make a new procedure type. Its parameter list is empty and its
//  value type is missing. We'll fill them in later.
func NewProcedureType() *ProcedureType {
	p := new(Parameter)
	return &ProcedureType{
		arity: 0,
		first: p,
		last:  p,
	}
}

//  ADD PARAMETER. Add a new parameter TYPE to the end of the parameter list.
func (p *ProcedureType) InsertParam(typ Typ) {
	p.arity++
	p.last.next = newParameter(typ)
	p.last = p.last.next
}

//  ADD VALUE. Add the value type.
func (p *ProcedureType) InsertValue(value Typ) {
	p.value = value
}

func (p *ProcedureType) Value() Typ {
	return p.value
}

//  GET ARITY. Return the number of parameters in this type.
func (p *ProcedureType) Arity() int {
	return p.arity
}

//  PARAMETER LIST. Return the parameter list of this type.
func (p *ProcedureType) ParameterList() *Parameter {
	return p.first.next
}

//  GET TYPE. Return the value type of this type.
func (p *ProcedureType) Type() token.Tokint {
	return p.value.Type()
}

//  TO STRING. Return a string that notates this type.
func (p *ProcedureType) String() string {
	return "PROC TYPE"
}
