package parser

import (
	"fmt"
	"log"

	"github.com/jackspirou/chip/types"

	"github.com/jackspirou/chip/ssa"
	"github.com/jackspirou/chip/parser/token"
)

// nextUnit parses a unit.
func (p *Parser) nextUnit() ssa.Node {
	p.enter()

	var regNode ssa.Node
	var reg *ssa.Register

	switch p.tok.Type {
	case token.IDENT:
		nameTok := p.tok
		fmt.Println(nameTok)
		p.next()
		if p.tok.Type == token.LPAREN {
			p.next() // '('
			if p.tok.Type != token.RPAREN {
				paramNode := p.nextExpression()
				fmt.Println(paramNode)
				for p.tok.Type == token.COMMA {
					p.next() // ','
					paramTok := p.tok
					fmt.Println(paramTok)
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

		reg = p.alloc.Request()
		// assembler.emit("li", reg, p.tok.String());

		p.next() // int constant

		regNode = ssa.NewRegNode(types.Int, reg)

	case token.FLOAT:
		// p.tok.String()
		p.next()
	case token.CHAR:
		// p.tok.String()
		p.next()
	case token.STRING:

		reg = p.alloc.Request()

		// Label label = global.enterString(scanner.getString());
		// assembler.emit("la", reg, label);

		p.next() // string constant

		regNode = ssa.NewRegNode(types.String, reg)

	default:
		log.Fatalf("term expected, got '%s'", p.tok)
	}

	p.exit()
	return regNode
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
