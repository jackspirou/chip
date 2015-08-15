package parser

// nextProcedure parses a procedure.
func (p *Parser) nextProcedure() {
	p.enter()
	p.scope.Open()
	p.nextProcedureSignature()
	p.nextProcedureBody()
	p.scope.Close()
	p.exit()
}
