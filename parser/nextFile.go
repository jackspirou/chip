package parser

// nextFile parses a source file.
func (p *Parser) nextFile() error {
	p.enterNext()

	p.nextPackage()
	p.nextImports()
	p.nextProgram()

	p.exitNext()
	return nil
}
