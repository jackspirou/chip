package parser

// Next File. Parse a source file.
func (p *Parser) nextFile() {
  p.enter()
  p.nextPackage()
  p.nextImports()
  p.nextProgram()
  p.exit()
}
