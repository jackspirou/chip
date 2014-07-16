package parser

// Next Procedure. Parse a procedure.
func (p *Parser) nextProcedure() {
	p.enter()
	p.scope.Open()
	p.nextProcedureSignature()
	p.nextProcedureBody()
	p.scope.Close()
	p.exit()
}
