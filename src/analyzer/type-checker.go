package analyzer

import (
	"fmt"

	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/analyzer/util"
	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/platform"
)

type typeChecker struct {
	*parseutil.Emitter

	platform platform.Platform

	// For personal sanity / simplicity, all register definitions with the same
	// name must have the same type, even through same name with different types
	// are allowed by ssa.
	nameType map[string]ast.Type
}

func CheckTypes(
	emitter *parseutil.Emitter,
	targetPlatform platform.Platform,
) util.Pass[ast.SourceEntry] {
	return &typeChecker{
		Emitter:  emitter,
		platform: targetPlatform,
		nameType: map[string]ast.Type{},
	}
}

// If immType is an literal type, return a largest corresponding real
// (int/float) type.  Otherwise, return the original type.
func (checker *typeChecker) convertImmediateType(
	immType ast.Type,
) ast.Type {
	switch immType.(type) {
	case *ast.PositiveIntLiteralType:
		return ast.NewU64(immType.StartEnd())
	case *ast.NegativeIntLiteralType:
		return ast.NewI64(immType.StartEnd())
	case *ast.FloatLiteralType:
		return ast.NewF64(immType.StartEnd())
	}
	return immType
}

func (checker *typeChecker) bindImmediateToType(
	value ast.Value,
	realType ast.Type,
) {
	switch imm := value.(type) {
	case *ast.IntImmediate:
		if imm.BindedType == nil {
			imm.BindedType = realType
		}
	case *ast.FloatImmediate:
		if imm.BindedType == nil {
			imm.BindedType = realType
		}
	}
}

func (checker *typeChecker) Process(entry ast.SourceEntry) {
	funcDef, ok := entry.(*ast.FunctionDefinition)
	if !ok {
		return
	}

	for _, def := range funcDef.AllParameters() {
		if def.Type == nil {
			panic("should never happen") // error previously emitted.
		}
		checker.nameType[def.Name] = def.Type
	}

	dfsOrder, _ := util.DFS(funcDef)
	for _, block := range dfsOrder {
		// Note: we don't need to check phi's source references since all
		// instruction destination definitions are eventually checked against
		// checker.nameType
		for _, phi := range block.Phis {
			phiType, ok := checker.nameType[phi.Dest.Name]
			if !ok { // an type error occurred previously
				phi.Dest.Type = ast.NewErrorType(phi.StartEndPos)
			} else {
				phi.Dest.Type = phiType
			}
		}

		for _, inst := range block.Instructions {
			evalType := checker.evaluateInstruction(inst)

			dest := inst.Destination()
			if dest != nil {
				checker.processDestination(dest, evalType)
				if !ast.IsErrorType(dest.Type) {
					for _, src := range inst.Sources() {
						// Backfill copy/non-conversion unary/binary operation immediate
						// sources' type.
						checker.bindImmediateToType(src, dest.Type)
					}
				}
				checker.checkRedefinition(dest)
			}
		}
	}
}

func (checker *typeChecker) evaluateInstruction(
	in ast.Instruction,
) ast.Type {
	switch inst := in.(type) {
	case *ast.CopyOperation:
		return inst.Src.Type()
	case *ast.UnaryOperation:
		return checker.evaluateUnaryOperation(inst)
	case *ast.BinaryOperation:
		return checker.evaluateBinaryOperation(inst)
	case *ast.FuncCall:
		switch inst.Kind {
		case ast.SysCall:
			return checker.evaluateSysCall(inst)
		case ast.Call:
			return checker.evaluateCall(inst)
		default:
			panic(fmt.Sprintf("unhandled func call kind (%s)", inst.Kind))
		}
	case *ast.Jump:
		return nil
	case *ast.ConditionalJump:
		checker.evaluateConditionalJump(inst)
		return nil
	case *ast.Terminal:
		checker.evaluateTerminal(inst)
		return nil
	default:
		panic(fmt.Sprintf("unhandled instruction type (%s)", in.Loc()))
	}
}

func (checker *typeChecker) evaluateUnaryOperation(
	inst *ast.UnaryOperation,
) ast.Type {
	opType := inst.Src.Type()
	if ast.IsErrorType(opType) {
		return opType
	}

	switch inst.Kind {
	case ast.ToI8, ast.ToI16, ast.ToI32, ast.ToI64,
		ast.ToU8, ast.ToU16, ast.ToU32, ast.ToU64,
		ast.ToF32, ast.ToF64:

		if ast.IsNumberSubType(opType) {
			opType = checker.convertImmediateType(opType)
			checker.bindImmediateToType(inst.Src, opType)
		}
	}

	switch inst.Kind {
	case ast.Neg:
		if ast.IsSignedIntSubType(opType) {
			return opType
		}
	case ast.Not:
		if ast.IsIntSubType(opType) {
			return opType
		}
	case ast.ToI8:
		if ast.IsNumberSubType(opType) {
			return ast.NewI8(inst.StartEnd())
		}
	case ast.ToI16:
		if ast.IsNumberSubType(opType) {
			return ast.NewI16(inst.StartEnd())
		}
	case ast.ToI32:
		if ast.IsNumberSubType(opType) {
			return ast.NewI32(inst.StartEnd())
		}
	case ast.ToI64:
		if ast.IsNumberSubType(opType) {
			return ast.NewI64(inst.StartEnd())
		}
	case ast.ToU8:
		if ast.IsNumberSubType(opType) {
			return ast.NewU8(inst.StartEnd())
		}
	case ast.ToU16:
		if ast.IsNumberSubType(opType) {
			return ast.NewU16(inst.StartEnd())
		}
	case ast.ToU32:
		if ast.IsNumberSubType(opType) {
			return ast.NewU32(inst.StartEnd())
		}
	case ast.ToU64:
		if ast.IsNumberSubType(opType) {
			return ast.NewU64(inst.StartEnd())
		}
	case ast.ToF32:
		if ast.IsNumberSubType(opType) {
			return ast.NewF32(inst.StartEnd())
		}
	case ast.ToF64:
		if ast.IsNumberSubType(opType) {
			return ast.NewF64(inst.StartEnd())
		}
	default:
		panic(fmt.Sprintf("unhandled unary operation kind (%s)", inst.Kind))
	}

	checker.Emit(
		inst.Src.Loc(),
		"cannot use type %s on unary operation (%s)",
		opType,
		inst.Kind)
	return ast.NewErrorType(opType.StartEnd())
}

func (checker *typeChecker) evaluateBinaryOperation(
	inst *ast.BinaryOperation,
) ast.Type {
	type1 := inst.Src1.Type()
	if ast.IsErrorType(type1) {
		return type1
	}

	type2 := inst.Src2.Type()
	if ast.IsErrorType(type2) {
		return type2
	}

	var opType ast.Type
	if type1.IsSubTypeOf(type2) {
		opType = type2
	} else if type2.IsSubTypeOf(type1) {
		opType = type1
	} else {
		checker.Emit(
			inst.Loc(),
			"binary operation cannot operate on different types: %s vs %s",
			type1,
			type2)
		return ast.NewErrorType(type1.StartEnd())
	}

	if ast.IsIntSubType(opType) {
		return opType
	}

	if ast.IsFloatSubType(opType) {
		switch inst.Kind {
		case ast.Add, ast.Sub, ast.Mul, ast.Div:
			return opType
		}
	}

	checker.Emit(
		opType.Loc(),
		"cannot use type %s on binary operation (%s)",
		opType,
		inst.Kind)
	return ast.NewErrorType(opType.StartEnd())
}

func (checker *typeChecker) evaluateSysCall(
	inst *ast.FuncCall,
) ast.Type {
	sysCallSpec := checker.platform.SysCallSpec()
	foundError := false
	funcValueType := checker.convertImmediateType(inst.Func.Type())
	if ast.IsErrorType(funcValueType) {
		foundError = true
	} else if !sysCallSpec.IsValidFuncValueType(funcValueType) {
		foundError = true
		checker.Emit(
			funcValueType.Loc(),
			"invalid syscall function value type, found %s",
			funcValueType)
	} else {
		checker.bindImmediateToType(inst.Func, funcValueType)
	}

	if len(inst.Args) > sysCallSpec.MaxNumberOfArgs() {
		foundError = true
		checker.Emit(inst.Loc(), "too many syscall arguments")
	}

	for idx, arg := range inst.Args {
		argType := checker.convertImmediateType(arg.Type())

		if ast.IsErrorType(argType) {
			foundError = true
			continue
		} else if !sysCallSpec.IsValidArgType(argType) {
			foundError = true
			checker.Emit(
				arg.Loc(),
				"invalid %d-th syscall argument type, found %s",
				idx,
				argType)
		} else {
			checker.bindImmediateToType(arg, argType)
		}
	}

	if foundError {
		return ast.NewErrorType(inst.StartEnd())
	}

	return sysCallSpec.ReturnType(inst.StartEnd())
}

func (checker *typeChecker) evaluateCall(
	inst *ast.FuncCall,
) ast.Type {
	fType := inst.Func.Type()
	if ast.IsErrorType(fType) {
		return fType
	}

	funcType, ok := fType.(*ast.FunctionType)
	if !ok {
		checker.Emit(
			fType.Loc(),
			"calling invalid function, expected func type, found %s",
			fType)
		return ast.NewErrorType(inst.StartEnd())
	}

	if len(funcType.ParameterTypes) != len(inst.Args) {
		checker.Emit(
			inst.Loc(),
			"invalid number of arguments pass to %s",
			funcType)
		return ast.NewErrorType(inst.StartEnd())
	}

	foundError := false
	for idx, paramType := range funcType.ParameterTypes {
		arg := inst.Args[idx]
		argType := arg.Type()
		if ast.IsErrorType(argType) {
			foundError = true
			continue
		}

		if !argType.IsSubTypeOf(paramType) {
			checker.Emit(
				arg.Loc(),
				"invalid %d-th argument, expected %s found %s",
				idx,
				paramType,
				argType)
			foundError = true
		} else {
			checker.bindImmediateToType(arg, paramType)
		}
	}

	if foundError {
		return ast.NewErrorType(inst.StartEnd())
	}
	return funcType.ReturnType
}

func (checker *typeChecker) evaluateConditionalJump(
	inst *ast.ConditionalJump,
) {
	type1 := inst.Src1.Type()
	type2 := inst.Src2.Type()
	if ast.IsErrorType(type1) || ast.IsErrorType(type2) {
		// Source dependencies have type check error
		return
	}

	var cmpType ast.Type
	if type1.IsSubTypeOf(type2) {
		cmpType = type2
	} else if type2.IsSubTypeOf(type1) {
		cmpType = type1
	} else {
		checker.Emit(
			inst.Loc(),
			"conditional jump cannot operate on different types: %s vs %s",
			type1,
			type2)
		return
	}

	switch inst.Kind {
	case ast.Jeq, ast.Jne:
		if !ast.IsComparableType(cmpType) {
			checker.Emit(
				inst.Loc(),
				"source type %s is not comparable",
				cmpType)
		}
	case ast.Jlt, ast.Jge:
		if !ast.IsOrderedType(cmpType) {
			checker.Emit(
				inst.Loc(),
				"source type %s is not ordered",
				cmpType)
		}
	default:
		panic(fmt.Sprintf("unhandled conditional jump kind (%s)", inst.Kind))
	}

	cmpType = checker.convertImmediateType(cmpType)
	checker.bindImmediateToType(inst.Src1, cmpType)
	checker.bindImmediateToType(inst.Src2, cmpType)

	return
}

func (checker *typeChecker) evaluateTerminal(
	inst *ast.Terminal,
) {
	switch inst.Kind {
	case ast.Ret:
		retType := inst.RetVal.Type()
		if ast.IsErrorType(retType) {
			return
		}

		funcRetType := inst.ParentBlock().ParentFuncDef.ReturnType
		if !retType.IsSubTypeOf(funcRetType) {
			checker.Emit(
				inst.Loc(),
				"invalid return value type %s, expected %s",
				retType,
				funcRetType)
		} else {
			checker.bindImmediateToType(inst.RetVal, funcRetType)
		}
	case ast.Exit:
		exitValueType := inst.RetVal.Type()
		if ast.IsErrorType(exitValueType) {
			return
		}

		if !checker.platform.SysCallSpec().IsValidExitArgType(exitValueType) {
			checker.Emit(
				inst.Loc(),
				"invalid exit value type, found %s",
				exitValueType)
		} else {
			checker.bindImmediateToType(
				inst.RetVal,
				checker.platform.SysCallSpec().ReturnType(inst.RetVal.StartEnd()))
		}
	default:
		panic(fmt.Sprintf("unhandled terminal kind (%s)", inst.Kind))
	}
	return
}

func (checker *typeChecker) processDestination(
	dest *ast.VariableDefinition,
	evalType ast.Type,
) {
	if dest.Type == nil {
		switch evalType.(type) {
		case *ast.PositiveIntLiteralType:
			checker.Emit(
				dest.Loc(),
				"cannot infer register type from int immediate, "+
					"destination (%s) must be explicitly typed",
				dest.Name)
			dest.Type = ast.NewErrorType(evalType.StartEnd())
		case *ast.NegativeIntLiteralType:
			checker.Emit(
				dest.Loc(),
				"cannot infer register type from int immediate, "+
					"destination (%s) must be explicitly typed",
				dest.Name)
			dest.Type = ast.NewErrorType(evalType.StartEnd())
		case *ast.FloatLiteralType:
			checker.Emit(
				dest.Loc(),
				"cannot infer register type from float immediate, "+
					"destination (%s) must be explicitly typed",
				dest.Name)
			dest.Type = ast.NewErrorType(evalType.StartEnd())
		default: // including ast.ErrorType
			dest.Type = evalType
		}
		return
	}

	switch evalType.(type) {
	case *ast.ErrorType:
	// Do nothing.  Assume definition's type is valid
	default:
		if !evalType.IsSubTypeOf(dest.Type) {
			checker.Emit(
				dest.Loc(),
				"cannot assign %s value to %s destination (%s)",
				evalType,
				dest.Type,
				dest.Name)
		}
	}
}

func (checker *typeChecker) checkRedefinition(
	dest *ast.VariableDefinition,
) {
	switch dest.Type.(type) {
	case *ast.PositiveIntLiteralType:
		panic(fmt.Sprintf("should never happen (%s)", dest.Loc()))
	case *ast.NegativeIntLiteralType:
		panic(fmt.Sprintf("should never happen (%s)", dest.Loc()))
	case *ast.FloatLiteralType:
		panic(fmt.Sprintf("should never happen (%s)", dest.Loc()))
	case *ast.ErrorType:
		return
	}

	prevType, ok := checker.nameType[dest.Name]
	if !ok {
		checker.nameType[dest.Name] = dest.Type
	} else if !prevType.Equals(dest.Type) {
		checker.Emit(
			dest.Loc(),
			"cannot redefine register (%s) with new type %s, "+
				"previous defined as %s at (%s)",
			dest.Name,
			dest.Type,
			prevType,
			prevType.Loc())
	}
}
