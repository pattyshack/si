package allocator

import (
	"strings"

	arch "github.com/pattyshack/chickadee/architecture"
)

type RegisterSelector struct {
	*BlockState

	selected   map[*arch.Register]struct{}
	assignment map[*arch.RegisterConstraint]*arch.Register
	scratch    *arch.Register
}

func NewRegisterSelector(block *BlockState) *RegisterSelector {
	return &RegisterSelector{
		BlockState: block,
		selected:   map[*arch.Register]struct{}{},
		assignment: map[*arch.RegisterConstraint]*arch.Register{},
	}
}

// This return if any of the location's registers is selected.
func (selector *RegisterSelector) IsSelected(loc *arch.DataLocation) bool {
	for _, reg := range loc.Registers {
		_, ok := selector.selected[reg]
		if ok {
			return true
		}
	}
	return false
}

func (selector *RegisterSelector) Reserve(
	register *arch.Register,
	constraint *arch.RegisterConstraint,
) {
	if register == nil || constraint == nil {
		panic("should never happen")
	}

	_, ok := selector.assignment[constraint]
	if ok {
		panic("should never happen")
	}

	selector.selected[register] = struct{}{}
	selector.assignment[constraint] = register
}

// TODO: use preference info for destination register selection
//
// Re-selecting the same constraint will return previously selected register
// (This is used for source / destination register sharing).
//
// When onlyUnusedRegister is true, select may return nil if no unused register
// satisfies the constraint.  When false, select may move data to free up a
// register that satisfies the constraint.
func (selector *RegisterSelector) Select(
	constraint *arch.RegisterConstraint,
	onlyUnusedRegister bool,
) *arch.Register {
	if constraint == nil {
		panic("should never happen")
	}

	register, ok := selector.assignment[constraint]
	if ok {
		return register
	}

	var free *arch.Register
	var candidate *arch.Register
	for _, regInfo := range selector.ValueLocations.Registers {
		_, ok = selector.selected[regInfo.Register]
		if ok {
			continue
		}

		if regInfo.UsedBy == nil {
			if constraint.SatisfyBy(regInfo.Register) {
				selector.Reserve(regInfo.Register, constraint)
				return regInfo.Register
			}

			free = regInfo.Register
		} else if candidate == nil && constraint.SatisfyBy(regInfo.Register) {
			candidate = regInfo.Register
		}
	}

	if onlyUnusedRegister {
		return nil
	}

	if candidate == nil {
		panic("should never happen")
	}

	if free == nil {
		// In theory, this only happens at the function entry point.
		// Call GetScratch/ReleaseScratch to force spilling callee-saved parameter.
		free = selector.SelectScratch()
		selector.ReleaseScratch(free)

		if constraint.SatisfyBy(free) {
			selector.Reserve(free, constraint)
			return free
		}

		// free != candidate since free doesn't Satisfy constraint.
	}

	selector.MoveRegister(candidate, free)
	selector.Reserve(candidate, constraint)
	return candidate
}

// By construction, there's always at least one unused register (this
// assumption is checked by the instruction constraints validator).  The
// function entry point is the only place where all registers could be in
// used; in this case, at least one of the register is a pseudo-source
// callee-saved register that is never used by the function.
func (selector *RegisterSelector) SelectScratch() *arch.Register {
	if selector.scratch != nil {
		panic("should never happen")
	}

	// The common case fast path

	for _, regInfo := range selector.ValueLocations.Registers {
		_, ok := selector.selected[regInfo.Register]
		if ok {
			continue
		}

		if regInfo.UsedBy == nil {
			selector.selected[regInfo.Register] = struct{}{}
			selector.scratch = regInfo.Register
			return regInfo.Register
		}
	}

	// Slow path to handle function entry point

	var candidate *RegisterInfo
	for _, regInfo := range selector.ValueLocations.Registers {
		_, ok := selector.selected[regInfo.Register]
		if ok {
			continue
		}

		defName := regInfo.UsedBy.Name

		// Registers holding real definitions are not eligible
		if !strings.HasPrefix(defName, "%") ||
			strings.HasPrefix(defName, "%%") {
			continue
		}

		// Previous frame pointer has highest spill priority
		if defName == arch.PreviousFramePointer {
			candidate = regInfo
			break
		}

		// Pick a register deterministically.  Any one will do.
		if candidate == nil || candidate.Index > regInfo.Index {
			candidate = regInfo
		}
	}

	if candidate == nil {
		panic("should never happen")
	}

	regLoc := candidate.UsedBy
	def, ok := selector.ValueLocations.ValueNames[regLoc.Name]
	if !ok {
		panic("should never happen")
	}

	stackLoc := selector.AllocateFixedStackLocation(def)
	selector.CopyLocation(regLoc, stackLoc, nil)
	selector.FreeLocation(regLoc)

	selector.selected[candidate.Register] = struct{}{}
	selector.scratch = candidate.Register
	return candidate.Register
}

func (selector *RegisterSelector) ReleaseScratch(register *arch.Register) {
	if selector.scratch != register {
		panic("should never happen")
	}

	delete(selector.selected, register)
	selector.scratch = nil
}
