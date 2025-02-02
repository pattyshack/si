package allocator

import (
	"strings"

	arch "github.com/pattyshack/chickadee/architecture"
)

type RegisterSelector struct {
	*BlockState

	// Exact match registers are not selectable by AnyGeneral/AnyFloat
	// constraint.
	exactMatch map[*arch.Register]struct{}

	// selectedSrc & selectedDest may overlap, scratch must be disjoint.

	selectedSrc map[*arch.Register]*arch.RegisterConstraint
	assignedSrc map[*arch.RegisterConstraint]*arch.Register

	selectedDest map[*arch.Register]*arch.RegisterConstraint
	assignedDest map[*arch.RegisterConstraint]*arch.Register

	scratch *arch.Register
}

func NewRegisterSelector(block *BlockState) *RegisterSelector {
	return &RegisterSelector{
		BlockState:   block,
		exactMatch:   map[*arch.Register]struct{}{},
		selectedSrc:  map[*arch.Register]*arch.RegisterConstraint{},
		assignedSrc:  map[*arch.RegisterConstraint]*arch.Register{},
		selectedDest: map[*arch.Register]*arch.RegisterConstraint{},
		assignedDest: map[*arch.RegisterConstraint]*arch.Register{},
	}
}

func (selector *RegisterSelector) ExactMatch(reg *arch.Register) {
	selector.exactMatch[reg] = struct{}{}
}

func (selector *RegisterSelector) isSelected(register *arch.Register) bool {
	_, ok := selector.selectedSrc[register]
	if ok {
		return true
	}

	_, ok = selector.selectedDest[register]
	if ok {
		return true
	}

	return register == selector.scratch
}

func (selector *RegisterSelector) ReserveSource(
	register *arch.Register,
	constraint *arch.RegisterConstraint,
) {
	if register == nil || constraint == nil {
		panic("should never happen")
	}

	if register == selector.scratch {
		panic("should never happen")
	}

	_, ok := selector.assignedSrc[constraint]
	if ok {
		panic("should never happen")
	}

	_, ok = selector.selectedSrc[register]
	if ok {
		panic("should never happen")
	}

	selector.selectedSrc[register] = constraint
	selector.assignedSrc[constraint] = register
}

func (selector *RegisterSelector) reserveDestination(
	register *arch.Register,
	constraint *arch.RegisterConstraint,
) {
	if register == nil || constraint == nil {
		panic("should never happen")
	}

	if register == selector.scratch {
		panic("should never happen")
	}

	_, ok := selector.assignedDest[constraint]
	if ok {
		panic("should never happen")
	}

	_, ok = selector.selectedDest[register]
	if ok {
		panic("should never happen")
	}

	selector.selectedDest[register] = constraint
	selector.assignedDest[constraint] = register
}

// When onlyUnusedRegister is true, select may return nil if no unused register
// satisfies the constraint.  When false, select may move data to free up a
// register that satisfies the constraint.
func (selector *RegisterSelector) selectRegister(
	constraint *arch.RegisterConstraint,
	onlyUnusedRegister bool,
	selected map[*arch.Register]*arch.RegisterConstraint,
	reserve func(*arch.Register, *arch.RegisterConstraint),
) *arch.Register {
	if constraint == nil || selector.scratch != nil {
		panic("should never happen")
	}

	isCandidate := func(reg *arch.Register) bool {
		if constraint.AnyGeneral || constraint.AnyFloat {
			_, ok := selector.exactMatch[reg]
			if ok {
				return false
			}
		}
		return constraint.SatisfyBy(reg)
	}

	var free *arch.Register
	var candidate *arch.Register
	for _, regInfo := range selector.ValueLocations.Registers {
		_, ok := selected[regInfo.Register]
		if ok {
			continue
		}

		if regInfo.UsedBy == nil {
			if isCandidate(regInfo.Register) {
				reserve(regInfo.Register, constraint)
				return regInfo.Register
			}

			free = regInfo.Register
		} else if candidate == nil && isCandidate(regInfo.Register) {
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

		if isCandidate(free) {
			reserve(free, constraint)
			return free
		}

		// free != candidate since free doesn't Satisfy constraint.
	}

	selector.MoveRegister(candidate, free)
	reserve(candidate, constraint)
	return candidate
}

func (selector *RegisterSelector) SelectSourceRegister(
	constraint *arch.RegisterConstraint,
	onlyUnusedRegister bool,
) *arch.Register {
	if len(selector.selectedDest) > 0 {
		panic("should never happen")
	}

	return selector.selectRegister(
		constraint,
		onlyUnusedRegister,
		selector.selectedSrc,
		selector.ReserveSource)
}

// TODO: use preference info for destination register selection
func (selector *RegisterSelector) SelectDestinationRegister(
	constraint *arch.RegisterConstraint,
) *arch.Register {
	register, ok := selector.assignedSrc[constraint]
	if ok {
		selector.reserveDestination(register, constraint)
		return register
	}

	return selector.selectRegister(
		constraint,
		false,
		selector.selectedDest,
		selector.reserveDestination)
}

func (selector *RegisterSelector) getFreeRegister() *arch.Register {
	for _, regInfo := range selector.ValueLocations.Registers {
		if regInfo.UsedBy != nil {
			continue
		}

		if selector.isSelected(regInfo.Register) {
			continue
		}

		return regInfo.Register
	}

	return nil
}

func (selector *RegisterSelector) SelectFreeRegister() *arch.Register {
	register := selector.getFreeRegister()
	if register == nil {
		panic("should never happen")
	}

	constraint := &arch.RegisterConstraint{
		Clobbered:  true,
		AnyGeneral: true,
		AnyFloat:   true,
	}

	selector.ReserveSource(register, constraint)
	selector.reserveDestination(register, constraint)
	return register
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

	register := selector.getFreeRegister()
	if register != nil {
		selector.scratch = register
		return register
	}

	// Slow path to handle function entry point

	var candidate *RegisterInfo
	for _, regInfo := range selector.ValueLocations.Registers {
		if selector.isSelected(regInfo.Register) {
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

	selector.scratch = candidate.Register
	return candidate.Register
}

func (selector *RegisterSelector) ReleaseScratch(register *arch.Register) {
	if selector.scratch != register {
		panic("should never happen")
	}

	selector.scratch = nil
}
