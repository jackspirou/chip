package parser

import (
	"errors"
	"fmt"

	"github.com/jackspirou/chip/ast"
	"github.com/jackspirou/chip/token"
)

// nextSum parses a sum.
func (p *Parser) nextSum() (sum ast.Node, err error) {
	p.enterNext()

	left, err := p.nextProduct()
	if err != nil {
		return nil, err
	}
	sum = left

	for p.tok.Type == token.ADD || p.tok.Type == token.SUB {

		// check(left, intType);
		if left.Type() != ast.INTEGER {
			return nil, errors.New("Expected Integer got " + string(left.Type()))
		}

		operator := ast.NewNode(ast.OPERATOR, p.tok, ast.String(p.tok.String()))

		p.next() // skip '+' or '-'

		/* rightDes := */
		right, err := p.nextProduct()
		if err != nil {
			return nil, err
		}

		// check(right, intType);
		if right.Type() != ast.INTEGER && right.Type() != ast.FLOAT {
			return nil, fmt.Errorf("Expected Integer got %s", string(right.Type()))
		}

		if operator.Token().Type == token.ADD {

			// this is just shananigans to make constant floats and ints work together
			if left.Type() == ast.FLOAT || right.Type() == ast.FLOAT {
				result := float64(left.IntegerValue()) + float64(right.IntegerValue()) + left.FloatValue() + right.FloatValue()
				fmt.Printf("Add: %f\n", result)
			} else {
				fmt.Printf("Add: %d\n", left.IntegerValue()+right.IntegerValue())
			}
			// left, err := strconv.Atoi(des.String())
			// if err != nil {
			// 	panic(err)
			// }
			// right, err := strconv.Atoi(rightDes.String())
			// if err != nil {
			// 	panic(err)
			// }

			// des := node.NewInt()
			// des.SetInt(left + right)
			// assembler.emit("add", left.getRegister(), left.getRegister(), right.getRegister());
		} else {
			// this is just shananigans to make constant floats and ints work together
			if left.Type() == ast.FLOAT || right.Type() == ast.FLOAT {
				result := float64(left.IntegerValue()) - float64(right.IntegerValue()) - left.FloatValue() - right.FloatValue()
				fmt.Printf("Add: %f\n", result)
			} else {
				fmt.Printf("Add: %d\n", left.IntegerValue()-right.IntegerValue())
			}

			// assembler.emit("sub", left.getRegister(), left.getRegister(), right.getRegister());
		}
		// allocator.release(right.getRegister());
		sum = right
	}

	p.exitNext()
	return sum, err
}
