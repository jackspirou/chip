package parser

// Next Procedure. Parse a procedure.
func (p *Parser) nextProcedure() {
  p.enter()
  p.nextProcedureSignature()
	p.nextProcedureBody()
  p.exit()
}
