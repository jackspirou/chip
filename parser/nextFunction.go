package parser

// nextFunction parses a function.
func (p *Parser) nextFunction() {
	p.enter()

	// open a new scope for this function declaration
	//
	// all varaibles in the function signature should be scoped to the
	// function body or basic block.
	p.scope.Open()

	p.nextFunctionSignature()
	p.nextFunctionBody()

	// close the scope to regain global scope access
	p.scope.Close()

	p.exit()
}
