//
// Tbv = Trust But Verify
// en.wikipedia.org/wiki/Trust,_but_verify
//
// Доверяй, но проверяй
// doveryai, no proveryai
// Trust, but verify
//

package scope

import (
	"errors"
	"fmt"
	"github.com/JackSpirou/chip/node"
	"github.com/JackSpirou/chip/typ"
)

type Tbv struct {
	table       map[string]node.Node
	impliedType typ.Typ
}

func NewMassie() *Tbv {
	t := new(Tbv)
	t.table = make(map[string]node.Node)
	return t
}

func (t *Tbv) ImplyType(imply typ.Typ) {
	t.impliedType = imply
}

func (t *Tbv) GetImpliedType() typ.Typ {
	return t.impliedType
}

func (t *Tbv) Trust(name string, node node.Node) {
	if _, ok := t.table[name]; ok {
		fmt.Println("'" + name + "' is already being trusted.")
	} else {
		t.table[name] = node
	}
}

func (t *Tbv) Check(name string, node node.Node) (bool, error) {
	if trustedNode, ok := t.table[name]; ok {

		trustedProcedureType := trustedNode.Type().(*typ.ProcedureType)
		procedureType := node.Type().(*typ.ProcedureType)

		// fmt.Println(trustedProcedureType.GetValue().GetType())

		if trustedProcedureType.Value() != nil {
			if trustedProcedureType.Value().Type() != procedureType.Value().Type() {
				return false, errors.New("The function '" + name + "' must return either a type " + trustedProcedureType.Value().String() + " or " + procedureType.Value().String() + ". Not both.")
			}
		}
		if trustedProcedureType.Arity() != procedureType.Arity() {
			return false, errors.New("Number of arguments do not match.")
		}
		trustedParameters := trustedProcedureType.ParameterList()
		nodeParameters := procedureType.ParameterList()
		for nodeParameters != nil {
			if nodeParameters.Type() != trustedParameters.Type() {
				return false, errors.New("Parameter types are off.")
			}
			nodeParameters = nodeParameters.Next()
			trustedParameters = trustedParameters.Next()
		}
		return true, nil
	}
	return false, nil
}

func (t *Tbv) Verify() {
	fmt.Println("VERIFYING....")
}
