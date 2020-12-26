package parser

import (
	"fmt"
	"log"
	"strconv"

	"github.com/jackspirou/chip/ast"
	"github.com/jackspirou/chip/token"
)

// nextUnit parses a unit.
func (p *Parser) nextUnit() (unit ast.Node, err error) {
	p.enterNext()

	switch p.tok.Type {
	case token.IDENT:
		// des = node.NewLabel(p.tok.String())
		unit = ast.NewNode(ast.IDENT, p.tok, ast.String(p.tok.String()))
		p.next()
		if p.tok.Type == token.LPAREN {
			p.next() // '('
			if p.tok.Type != token.RPAREN {
				p.nextExpression()
				for p.tok.Type == token.COMMA {
					p.next() // ','
					p.nextExpression()
				}
			}
			p.nextExpected(token.RPAREN)
		} else if p.tok.Type == token.LBRACK {
			p.next()
			p.nextExpression()
			p.nextExpected(token.RBRACK)
		}
	case token.INT:
		// convert token from string to integer
		i, err := strconv.Atoi(p.tok.String())
		if err != nil {
			return nil, err
		}

		// make a new integer node
		unit = ast.NewNode(ast.INTEGER, p.tok, ast.Integer(i))

		p.next()
	case token.FLOAT:
		f, err := strconv.ParseFloat(p.tok.String(), 64)
		if err != nil {
			return nil, err
		}

		// make a new integer node
		unit = ast.NewNode(ast.FLOAT, p.tok, ast.Float(f))

		p.next()
	case token.CHAR:
		// p.tok.String()
		p.next()
	case token.STRING:

		unit = ast.NewNode(ast.STRING, p.tok, ast.String(p.tok.String()))

		// reg = p.alloc.Request()

		// Label label = global.enterString(scanner.getString());
		// assembler.emit("la", reg, label);

		p.next() // string constant

		// regNode = node.NewRegNode(types.String, reg)

	default:
		log.Fatalf("term expected, got '%s'", p.tok)
	}

	fmt.Println("UNIT: ", unit, err)

	p.exitNext()
	return unit, err
}

/*
// Parses a unit.
 private RegisterDescriptor nextUnit()
 {
		 enter("nextUnit");

		 RegisterDescriptor descriptor = null;
		 Allocator.Register reg = null;

		 switch (scanner.getToken())
		 {
				 case intConstantToken:

						 reg = allocator.request();
						 assembler.emit("li", reg, scanner.getInt());

						 scanner.nextToken(); // Skip int constant token.

						 descriptor = new RegisterDescriptor(intType, reg);

						 break;

				 case stringConstantToken:


						 reg = allocator.request();
						 Label label = global.enterString(scanner.getString());
						 assembler.emit("la", reg, label);

						 scanner.nextToken(); // Skip string constant token.

						 descriptor = new RegisterDescriptor(stringType, reg);

						 break;

				 case openParenToken:
						 scanner.nextToken(); // Skip '(' token.

						 descriptor = nextExpression();

						 nextExpected(closeParenToken);
						 break;

				 case nameToken:
						 nextExpected(nameToken);

						 switch (scanner.getToken())
						 {
								 case openParenToken:
								 {
										 descriptor = nextCall();
										 break;
								 }

								 case openBracketToken:
								 {
										 NameDescriptor nameDes = symbolTable.getDescriptor(scanner.getString());
										 reg = nameDes.rvalue();

										 descriptor = new RegisterDescriptor(nameDes.getType(), reg);

										 if(! (descriptor.getType() instanceof ArrayType))
												 source.error(scanner.getString() +
														 " is not an array.");

										 scanner.nextToken(); // Skip '[' token.
										 descriptor = nextExpression();
										 check(descriptor, intType);

										 nextExpected(closeBracketToken);

										 assembler.emit("sll", descriptor.getRegister(), descriptor.getRegister(), 2);
										 assembler.emit("add", reg, reg, descriptor.getRegister());
										 assembler.emit("lw", reg, 0, reg);

										 allocator.release(descriptor.getRegister());


										 descriptor = new RegisterDescriptor(intType, reg);
										 break;
								 }
								 default:
								 {
										 NameDescriptor nameDes = symbolTable.getDescriptor(scanner.getString());
										 descriptor = new RegisterDescriptor(nameDes.getType(), nameDes.rvalue());
										 break;
								 }
						 }

						 break;
				 default:
						 source.error("Unit expected.");
						 break;
		 }

		 exit("nextUnit");

		 return descriptor;
 }
*/
