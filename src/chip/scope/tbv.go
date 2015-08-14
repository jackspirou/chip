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
	table map[string]node.Node
	typ   types.Typer
}

// NewTBV returns a new TBV object.
func NewTBV() *TBV {
	t := new(TBV)
	t.table = make(map[string]node.Node)
	return t
}

func (t *TBV) Imply(typ types.Typer) {
	t.typ = typ
}

func (t *TBV) Type() types.Typer {
	return t.typ
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

		trustedProc := trustedNode.Type().(*types.Proc)
		proc := node.Type().(*types.Proc)

		if trustedProc.Value() != nil {
			if trustedProc.Value().Type() != proc.Value().Type() {
				return false, errors.New("The function '" + name + "' must return either a type " + trustedProc.Value().String() + " or " + proc.Value().String() + ". Not both.")
			}
		}
		if trustedProc.Arity() != proc.Arity() {
			return false, errors.New("Number of arguments do not match.")
		}
		trustedParam := trustedProc.Param()
		param := proc.Param()
		for param != nil {
			if param.Type() != trustedParam.Type() {
				return false, errors.New("Parameter types are off.")
			}
			param = param.Next()
			trustedParam = trustedParam.Next()
		}
		return true, nil
	}
	return false, nil
}

func (t *TBV) Verify() {
	fmt.Println("VERIFYING....")
}
