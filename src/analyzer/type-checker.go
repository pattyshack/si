package analyzer

import (
	"fmt"

	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
)

type typeChecker struct {
	*parseutil.Emitter

	// For personal sanity / simplicity, all register definitions with the same
	// name must have the same type, even through same name with different types
	// are allowed by ssa.
	nameType map[string]ast.Type
}

func CheckTypes(
	emitter *parseutil.Emitter,
) Pass[ast.SourceEntry] {
	return &typeChecker{
		Emitter:  emitter,
		nameType: map[string]ast.Type{},
	}
}

func (checker *typeChecker) Process(entry ast.SourceEntry) {
	funcDef, ok := entry.(*ast.FuncDefinition)
	if !ok {
		return
	}

	for _, def := range funcDef.Parameters {
		if def.Type == nil {
			panic("should never happen") // error previously emitted.
		}
		checker.nameType[def.Name] = def.Type
	}

	processed := map[*ast.Block]struct{}{}
	queue := []*ast.Block{funcDef.Blocks[0]}
	for len(queue) > 0 {
		block := queue[0]
		queue = queue[1:]

		_, ok := processed[block]
		if ok {
			continue
		}
		processed[block] = struct{}{}

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
			evalType := checker.evaluateInstruction(inst, funcDef.ReturnType)

			dest := inst.Destination()
			if dest != nil {
				checker.processDestination(dest, evalType)
				checker.checkRedefinition(dest)
			}
		}

		queue = append(queue, block.Children...)
	}
}

func (checker *typeChecker) evaluateInstruction(
	in ast.Instruction,
	funcRetType ast.Type,
) ast.Type {
	switch inst := in.(type) {
	case *ast.AssignOperation:
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
		return checker.evaluateConditionalJump(inst)
	case *ast.Terminal:
		return checker.evaluateTerminal(inst, funcRetType)
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

	if ast.IsIntSubType(opType) {
		return opType
	}

	if ast.IsFloatSubType(opType) {
		switch inst.Kind {
		case ast.Neg:
			return opType
		}
	}

	checker.Emit(
		inst.Loc(),
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
		inst.Loc(),
		"cannot use type %s on binary operation (%s)",
		opType,
		inst.Kind)
	return ast.NewErrorType(opType.StartEnd())
}

func (checker *typeChecker) evaluateSysCall(
	inst *ast.FuncCall,
) ast.Type {
	panic("TODO syscall specification")
}

func (checker *typeChecker) evaluateCall(
	inst *ast.FuncCall,
) ast.Type {
	fType := inst.Func.Type()
	if ast.IsErrorType(fType) {
		return fType
	}

	funcType, ok := fType.(ast.FunctionType)
	if !ok {
		checker.Emit(
			inst.Loc(),
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
		argType := inst.Args[idx].Type()
		if ast.IsErrorType(argType) {
			foundError = true
			continue
		}

		if !argType.IsSubTypeOf(paramType) {
			checker.Emit(
				inst.Loc(),
				"invalid %d-th argument, expected %s found %s",
				idx,
				paramType,
				argType)
			foundError = true
		}
	}

	if foundError {
		return ast.NewErrorType(inst.StartEnd())
	}
	return funcType.ReturnType
}

func (checker *typeChecker) evaluateConditionalJump(
	inst *ast.ConditionalJump,
) ast.Type {
	type1 := inst.Src1.Type()
	type2 := inst.Src2.Type()
	if ast.IsErrorType(type1) || ast.IsErrorType(type2) {
		// Source dependencies have type check error
		return nil
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
		return nil
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

	return nil
}

func (checker *typeChecker) evaluateTerminal(
	inst *ast.Terminal,
	funcRetType ast.Type,
) ast.Type {
	switch inst.Kind {
	case ast.Ret:
		retType := inst.Src.Type()
		if ast.IsErrorType(retType) {
			return nil
		}

		if !retType.IsSubTypeOf(funcRetType) {
			checker.Emit(
				inst.Loc(),
				"invalid return value type %s, expected %s",
				retType)
		}
	case ast.Exit:
		retType := inst.Src.Type()
		if ast.IsErrorType(retType) {
			return nil
		}

		panic("TODO exit syscall specification")

	default:
		panic(fmt.Sprintf("unhandled terminal kind (%s)", inst.Kind))
	}
	return nil
}

func (checker *typeChecker) processDestination(
	dest *ast.RegisterDefinition,
	evalType ast.Type,
) {
	if dest.Type == nil {
		switch evalType.(type) {
		case ast.IntLiteralType:
			checker.Emit(
				dest.Loc(),
				"cannot infer register type from int immediate, "+
					"destination (%s) must be explicitly typed",
				dest.Name)
			dest.Type = ast.NewErrorType(evalType.StartEnd())
		case ast.FloatLiteralType:
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
	case ast.ErrorType:
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
	dest *ast.RegisterDefinition,
) {
	switch dest.Type.(type) {
	case ast.IntLiteralType:
		panic(fmt.Sprintf("should never happen (%s)", dest.Loc()))
	case ast.FloatLiteralType:
		panic(fmt.Sprintf("should never happen (%s)", dest.Loc()))
	case ast.ErrorType:
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
