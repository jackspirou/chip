package parser

// nextFile parses a source file.
func (p *Parser) nextFile() error {
	p.enter()
	p.nextPackage()
	p.nextImports()
	p.nextProgram()
	p.exit()
	return nil
}
