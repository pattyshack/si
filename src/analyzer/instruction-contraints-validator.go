package analyzer

/*
TODO redo everything ...

import (
	"fmt"

	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/architecture"
	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/platform"
)

type instructionConstraintsValidator struct {
	platform platform.Platform
}

func ValidateInstructionConstraints(
	targetPlatform platform.Platform,
) Pass[ast.SourceEntry] {
	return &instructionConstraintsValidator{
		platform: targetPlatform,
	}
}

func (validator *instructionConstraintsValidator) Process(
	entry ast.SourceEntry,
) {
	funcDef, ok := entry.(*ast.FunctionDefinition)
	if !ok {
		return
	}

	validator.Validate(funcDef.Loc(), funcDef.CallRetConstraints, true, false)
	for _, block := range funcDef.Blocks {
		for _, inst := range block.Instructions {
			isCallConvention := false
			isControlFlowInstruction := false
			switch inst.(type) {
			case *ast.FuncCall:
				isCallConvention = true
			case *ast.Terminal:
				isCallConvention = true
				isControlFlowInstruction = true
			case ast.ControlFlowInstruction:
				isControlFlowInstruction = true
			}

			validator.Validate(
				inst.Loc(),
				validator.platform.InstructionConstraints(inst),
				isCallConvention,
				isControlFlowInstruction)
		}
	}
}

func (validator *instructionConstraintsValidator) Validate(
	pos parseutil.Location,
	constraints *architecture.InstructionConstraints,
	isCallConvention bool,
	isControlFlowInstruction bool,
) {
	if !isCallConvention {
		if constraints.FuncValue != nil {
			panic(fmt.Sprintf("invalid: %s", pos))
		}
		if len(constraints.PseudoSources) > 0 {
			panic(fmt.Sprintf("invalid: %s", pos))
		}
		if len(constraints.SrcStackLocations) != 0 {
			panic(fmt.Sprintf("invalid: %s", pos))
		}
		if constraints.DestStackLocation != nil {
			panic(fmt.Sprintf("invalid: %s", pos))
		}
		if isControlFlowInstruction {
			if constraints.Destination != nil {
				panic(fmt.Sprintf("invalid: %s", pos))
			}
		} else if constraints.Destination == nil {
			panic(fmt.Sprintf("invalid: %s", pos))
		}
	} else {
		if constraints.FuncValue == nil {
			panic(fmt.Sprintf("invalid: %s", pos))
		}
		if constraints.Destination == nil {
			panic(fmt.Sprintf("invalid: %s", pos))
		}
	}

	// A required register may only appear once in the source list (including
	// pseudo sources) and func value.  The same required register may
	// reappear once in the destintation to indicate register reuse.
	requiredSrcRegisters := map[*architecture.Register]bool{}
	requiredDestRegisters := map[*architecture.Register]bool{}

	collectRequired := func(
		loc *architecture.LocationConstraint,
		required map[*architecture.Register]bool,
	) {
		for _, reg := range loc.Registers {
			if reg.Require == nil {
				continue
			}
			_, ok := required[reg.Require]
			if ok {
				panic(fmt.Sprintf("invalid: %s", pos))
			}
			required[reg.Require] = reg.Clobbered
		}
	}

	if constraints.FuncValue != nil {
		validator.ValidateLocation(pos, constraints.FuncValue, isCallConvention)
		if len(constraints.FuncValue.Registers) != 1 {
			panic(fmt.Sprintf("invalid: %s", pos))
		}
		collectRequired(constraints.FuncValue, requiredSrcRegisters)
	}

	for _, src := range constraints.Sources {
		validator.ValidateLocation(pos, src, isCallConvention)
		collectRequired(src, requiredSrcRegisters)
	}

	for _, src := range constraints.PseudoSources {
		validator.ValidateLocation(pos, src, isCallConvention)
		if len(src.Registers) != 1 {
			panic(fmt.Sprintf("invalid: %s", pos))
		}
		if src.Registers[0].Clobbered {
			panic(fmt.Sprintf("invalid: %s", pos))
		}
		collectRequired(src, requiredSrcRegisters)
	}

	if constraints.Destination != nil {
		validator.ValidateLocation(pos, constraints.Destination, isCallConvention)
		collectRequired(constraints.Destination, requiredDestRegisters)
	}

	for reg, destClobbered := range requiredDestRegisters {
		srcClobbered, ok := requiredSrcRegisters[reg]
		if ok && srcClobbered != destClobbered {
			panic(fmt.Sprintf("invalid: %s", pos))
		}
	}
}

func (validator *instructionConstraintsValidator) ValidateLocation(
	pos parseutil.Location,
	constraint *architecture.LocationConstraint,
	isCallConvention bool,
) {
	if constraint.AnyLocation {
		if constraint.RequireOnStack {
			panic(fmt.Sprintf("invalid: %s", pos))
		}
		if len(constraint.Registers) > 0 {
			panic(fmt.Sprintf("invalid: %s", pos))
		}
		if isCallConvention {
			panic(fmt.Sprintf("invalid: %s", pos))
		}
		// Size could be zero
	} else if constraint.RequireOnStack {
		if len(constraint.Registers) > 0 {
			panic(fmt.Sprintf("invalid: %s", pos))
		}
		if constraint.Size == 0 {
			panic(fmt.Sprintf("invalid: %s", pos))
		}
		if !isCallConvention {
			panic(fmt.Sprintf("invalid: %s", pos))
		}
	} else {
		for _, reg := range constraint.Registers {
			if reg.AnyGeneral || reg.AnyFloat {
				if reg.Require != nil {
					panic(fmt.Sprintf("invalid: %s", pos))
				}
				if isCallConvention {
					panic(fmt.Sprintf("invalid: %s", pos))
				}
			} else if reg.Require == nil {
				panic(fmt.Sprintf("invalid: %s", pos))
			}
		}
	}
}
*/
