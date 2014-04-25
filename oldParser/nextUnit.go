package oldParser

import (
	// "fmt"
	"jak/common"
	"jak/generator"
	"jak/typ"
	"strconv"
)

// Next Unit. Parse a unit.
func nextUnit() *RegisterDescriptor {
	common.Enter("nextUnit")
	switch scan.GetToken() {
	case common.NameToken:
		name := scan.GetString()
		scan.NextToken() // Skip the nameToken.
		nameDes, err := symboltable.GetDescriptor(name)
		if scan.GetToken() == common.OpenParenToken { // A procedure.
			scan.NextToken() // Skip the '('.
			var des NameDescriptor
			var procedureType *typ.ProcedureType
			var label *generator.Label
			if nameDes == nil { // The procedure has not been previously defined.  We must gather as much info as possible for Massie.
				procedureType = typ.NewProcedureType()
				procedureType.AddValue(massie.GetImpliedType())
				label = generator.NewLabel(name)
			} else { // The procedure has been previously defined.
				des = nameDes.(NameDescriptor) // Assert that 'des' is a NameDescriptor.
				// checkProcedureType(des);
				procedureType = des.GetType().(*typ.ProcedureType)
				label = des.GetLabel()
			}
			if scan.GetToken() != common.CloseParenToken {
				regDes := nextExpression()
				// checkType(procedureType.getParameters(), regDes.getType());
				numOfArgs := 1
				generate.Comment("Next Unit")
				/*
				* STORE WORD STUFF WAS HERE (SW)
				 */
				if nameDes == nil {
					procedureType.AddParameter(regDes.GetType())
				}
				allocate.Release(regDes.GetRegister())
				for scan.GetToken() == common.CommaToken {
					scan.NextToken() // Skip commaToken.
					regDes = nextExpression()
					// checkType(regDes, procedureType.getParameters().getNext().getType());
					numOfArgs++
					generate.Comment("Next Unit")
					/*
					* STORE WORD STUFF WAS HERE (SW)
					 */
					if nameDes == nil {
						procedureType.AddParameter(regDes.GetType())
					}
					allocate.Release(regDes.GetRegister())
				}
				if numOfArgs > procedureType.GetArity() {
					scan.Source.Error("Too many arguments for this procedure.")
				} else if numOfArgs < procedureType.GetArity() {
					scan.Source.Error("Missing some of the procedure's arguments.")
				}
				nextExpected(common.CloseParenToken)
				generate.Comment("Next Unit")
				generate.Jump(common.OP_JAL, label)
				reg := allocate.Request()
				// generate.Imm(common.OP_MOV, reg.String(), allocate.V0.String())
				if nameDes == nil {
					des = NewLabelDescriptor(procedureType, label)
					massie.Trust(name, des)
					symboltable.SetGlobalDescriptor(name, des)
				}
				tempReg := NewRegisterDescriptor(procedureType.GetValue(), reg)
				common.Exit("nextUnit")
				return tempReg
			} else {
				if procedureType.GetArity() != 0 {
					scan.Source.Error("Missing some of the procedure's arguments.")
				}
				nextExpected(common.CloseParenToken) // Skip closeParenToken.
				generate.Comment("Next Unit")
				generate.Jump(common.OP_JAL, label)
				reg := allocate.Request()
				// generate.Imm(common.OP_MOV, reg.String(), allocate.V0.String())
				tempReg := NewRegisterDescriptor(procedureType.GetValue(), reg)
				common.Exit("nextUnit")
				return tempReg
			}
		} else if scan.GetToken() == common.OpenBracketToken {
			check(err)
			scan.NextToken() // Skip openBracketToken.
			reg := allocate.Request()
			//
			// allocate.Register reg = nameDes.rvalue(); instead of reg := nameDes.GetRegister() above
			//
			indexDes := nextExpression()
			// checkArrayType(nameDes);
			// checkIntType(indexDes);
			nextExpected(common.CloseBracketToken)
			generate.Comment("Next Unit")
			generate.Branch(common.OP_SLL, indexDes.GetRegister().String(), indexDes.GetRegister().String(), "2")
			generate.Branch(common.OP_ADD, reg.String(), reg.String(), indexDes.GetRegister().String())
			allocate.Release(indexDes.GetRegister())
			// assembler.emit("lw", reg, 0, reg);
			common.Exit("nextUnit")
			tempReg := NewRegisterDescriptor(intType, reg)
			return tempReg
		} else {
			check(err)
			generate.Comment("Next Unit Code")
			// des := nameDes.(*RegisterDescriptor)
			// reg := des.GetRegister()
			reg := allocate.Request()
			tempReg := NewRegisterDescriptor(nameDes.GetType(), reg)
			common.Exit("nextUnit")
			return tempReg
		}
	case common.OpenParenToken:
		scan.NextToken() // Skip '('.
		des := nextExpression()
		nextExpected(common.CloseParenToken)
		common.Exit("nextUnit")
		return des
	case common.IntConstantToken:
		intValue := scan.GetInteger()
		scan.NextToken() // Skip intConstantToken.
		reg := allocate.Request()
		generate.Comment("Next Unit")
		generate.Imm(common.OP_LI, reg.String(), strconv.Itoa(intValue))
		des := NewRegisterDescriptor(intType, reg)
		common.Exit("nextUnit")
		return des
	case common.StringConstantToken:
		// (NEEDS TO BE FIXED) stringValue := scan.GetString()
		scan.NextToken() // Skip stringConstantToken.
		reg := allocate.Request()
		// Label label = global.enterString(stringValue);
		generate.Comment("Next Unit")
		// assembler.emit("la", reg, label);
		des := NewRegisterDescriptor(stringType, reg)
		common.Exit("nextUnit")
		return des
	default:
		scan.Source.Error("Unit expected.")
	}
	panic("Satisfy Return")
}
