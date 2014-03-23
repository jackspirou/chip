//
// SNARL/PASS TWO. Performs the second pass of the Snarl compiler.
//
//		Jack Spirou
//		1 May 2012
//

// PASS TWO. Performs the second pass of the Snarl compiler.

package parser

import (
	"jak/common"
	"jak/generator"
	"jak/scanner"
	"jak/typ"
)

var scan *scanner.Scanner    // Integer value for the current token.
var symboltable *SymbolTable //
var massie *Massie
var generate *generator.Generator //
var allocate *generator.Allocator // Allocator.
var intType *typ.BasicType
var stringType *typ.BasicType

func Parse(path string) {
	source := scanner.NewSource(path)
	scan = scanner.NewScanner(source)
	symboltable = NewSymbolTable()
	symboltable.Push()
	massie = NewMassie()
	allocate = generator.NewAllocator()
	generate = generator.NewGenerator(path)
	intType = typ.NewBasicType(common.INT)
	stringType = typ.NewBasicType(common.STRING)
	// p.global = NewGlobal()
	nextProgram()
}

// Next Program. Parse a source file.
func nextProgram() {
	common.Enter("nextProgram")
	generate.Emit(".text")
	for scan.GetToken() != common.EndFileToken {
		if scan.GetToken() == common.BoldFuncToken {
			nextProcedure()
		} else {
			nextDeclaration()
		}
	}
	common.Exit("nextProgram")
}
