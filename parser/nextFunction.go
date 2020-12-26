package parser

// nextFunction parses a function.
func (p *Parser) nextFunction() {
	p.enterNext()

	tok := p.tok
	_, err := p.scope.Lookup(tok)
	if err != nil {
		userErr(err, tok)
	}

	p.exitNext()
}

/*
   	// Retrieve the ProcedureType from the symbolTable.
   	Descriptor descriptor = symbolTable.getDescriptor(scanner.getString());
   	ProcedureType type = null;

   if(descriptor.getType() instanceof ProcedureType)
   		type = (ProcedureType)descriptor.getType();
   else
   		source.error(scanner.getString() + " is not a procedure.");

   // Has to be a GlobalProcedureDescriptor at this point.
   GlobalProcedureDescriptor procedureDescriptor = (GlobalProcedureDescriptor) descriptor;

   int arity = 0;
   scanner.nextToken(); // Skip '(' token.
   if (scanner.getToken() != closeParenToken)
   {
   		arity++;
   		if(type.getArity() < arity)
   				source.error("Invalid number of arguments.");

   		ProcedureType.Parameter param = type.getParameters();
   		RegisterDescriptor regdes = nextExpression();
   		check(regdes, param.getType());

   		// Compile code for first expression in call.
   		assembler.emit("# Call.");
   		assembler.emit("sw", regdes.getRegister(), 0, allocator.sp);
   		assembler.emit("addi", allocator.sp, allocator.sp, -4);
   		allocator.release(regdes.getRegister());

   		while (scanner.getToken() == commaToken)
   		{
   				nextExpected(commaToken, ", or ) expected.");

   				arity++;
   				if(type.getArity() < arity)
   						source.error("Invalid number of arguments.");

   				regdes = nextExpression();
   				param = param.getNext();
   				check(regdes, param.getType());

   				// Compile code for E sub k.
   				assembler.emit("sw", regdes.getRegister(), 0, allocator.sp);
   				assembler.emit("addi", allocator.sp, allocator.sp, -4);
   				allocator.release(regdes.getRegister());
   		}
   }
   nextExpected(closeParenToken);

   if(type.getArity() != arity)
   		source.error("Invalid number of arguments.");

   // Compile code for jump.
   Allocator.Register reg = allocator.request();
   assembler.emit("jal", procedureDescriptor.getLabel());
   assembler.emit("move", reg, allocator.v0);

   exit("nextCall");

   return new RegisterDescriptor(type.getValue(), reg);
*/
