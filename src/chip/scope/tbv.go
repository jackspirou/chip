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

	"github.com/jackspirou/chip/src/chip/node"
	"github.com/jackspirou/chip/src/chip/types"
)

// TBV stands for "Доверяй, но проверяй" or "Trust, but verify."
//
// TBV records a table of nodes and types so that parameters and return types
// can be trusted/executed durring recursive decent parsing, but also verified.
type TBV struct {
	table       map[string]node.Node
	impliedType types.Typer
}

// NewTBV returns a new TBV object.
func NewTBV() *TBV {
	t := new(TBV)
	t.table = make(map[string]node.Node)
	return t
}

func (t *TBV) Imply(t types.Typer) {
	t.impliedType = imply
}

func (t *TBV) Implied() types.Typer {
	return t.impliedType
}

func (t *TBV) Trust(name string, node node.Node) {
	if _, ok := t.table[name]; ok {
		fmt.Println("'" + name + "' is already being trusted.")
	} else {
		t.table[name] = node
	}
}

func (t *TBV) Check(name string, node node.Node) (bool, error) {
	if trustedNode, ok := t.table[name]; ok {

		trustedProcedureType := trustedNode.Type().(*types.ProcedureType)
		procedureType := node.Type().(*types.ProcedureType)

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

func (t *TBV) Verify() {
	fmt.Println("VERIFYING....")
}
