//
// TBV = Trust But Verify
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

	"github.com/jackspirou/chip/token"

	"github.com/jackspirou/chip/node"
	"github.com/jackspirou/chip/types"
)

// TBV stands for "Доверяй, но проверяй" or "Trust, but verify."
//
// TBV records a table of nodes and types so that parameters and return types
// can be trusted/executed durring recursive decent parsing, but also verified.
type TBV struct {
	table map[string]node.Node
}

// NewTBV returns a new TBV object.
func NewTBV() *TBV {
	return &TBV{table: make(map[string]node.Node)}
}

// Contains checks if a token name appears in TBV table.
func (t TBV) Contains(token fmt.Stringer) bool {
	name := token.String()
	_, ok := t.table[name]
	return ok
}

// Trust takes a name token and node that has not yet been defined and "trusts"
// that it will be defined later according to the node type values that were
// implied by the parser.
func (t *TBV) Trust(token fmt.Stringer, node node.Node) {
	name := token.String()
	if _, ok := t.table[name]; ok {
		// p.enter()("'" + name + "' is already being trusted.")
	} else {
		t.table[name] = node
	}
}

// Verify verifies that the provided token and node match with a corresponding
// pair that were trusted previously.
func (t *TBV) Verify(tok fmt.Stringer, node node.Node) (bool, error) {

	name := tok.String()

	// check if the node is present in the TBV table
	if trustNode, ok := t.table[name]; ok {

		if trustNode.Type() != node.Type() {
			return true, errors.New("mismatch types")
		}

		switch trustNode.Type().Token() {
		case token.FUNC:

			trustFuncType := trustNode.Type().(*types.Func)
			funcType := node.Type().(*types.Func)

			if trustFuncType.Value() != nil {
				if trustFuncType.Value() != funcType.Value() {
					return false, errors.New("The function '" + name + "' must return either a type " + trustFuncType.Value().String() + " or " + funcType.Value().String() + ". Not both.")
				}
			}
			if trustFuncType.Arity() != funcType.Arity() {
				return false, errors.New("Number of arguments do not match.")
			}
			trustParam := trustFuncType.Param()
			param := funcType.Param()
			for param != nil {
				if param != trustParam {
					return false, errors.New("Parameter types are off.")
				}
				param = param.Next()
				trustParam = trustParam.Next()
			}
			return true, nil
		}
	}
	return false, nil
}
