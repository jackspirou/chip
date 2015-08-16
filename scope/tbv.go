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

	"github.com/jackspirou/chip/node"
	"github.com/jackspirou/chip/types"
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
	return &TBV{table: make(map[string]node.Node)}
}

// Imply something about a type. I say 'something' because I forgot why this
// was written to begin with. I'll remove it later... hopefully.
func (t *TBV) Imply(typ types.Typer) {
	t.typ = typ
}

// Type returns the type that was implied. If the Imply method is removed, so
// should this method be removed.
func (t TBV) Type() types.Typer {
	return t.typ
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
func (t *TBV) Verify(token fmt.Stringer, node node.Node) (bool, error) {

	name := token.String()

	// is the node is present in the TBV table
	if trustedNode, ok := t.table[name]; ok {

		trustedFunc := trustedNode.Type().(*types.Func)
		fn := node.Type().(*types.Func)

		if trustedFunc.Value() != nil {
			if trustedFunc.Value().Type() != fn.Value().Type() {
				return false, errors.New("The function '" + name + "' must return either a type " + trustedFunc.Value().String() + " or " + fn.Value().String() + ". Not both.")
			}
		}
		if trustedFunc.Arity() != fn.Arity() {
			return false, errors.New("Number of arguments do not match.")
		}
		trustedParam := trustedFunc.Param()
		param := fn.Param()
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
