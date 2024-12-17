package allocator

import (
	"github.com/pattyshack/chickadee/ast"
)

// All distances are in terms of instruction distance relative to the beginning
// of the block (phis are at distance zero).
type LiveRange struct {
	Start int // first live distance (relative to beginning of current block)
	End   int // last use distance, inclusive.
}

// The block's execution state at a particular point in time.
type BlockState struct {
	*ast.Block

	LiveIn  LiveSet
	LiveOut LiveSet

	NextInstIdx int

	// Note: unused definitions are not included in LiveRanges
	LiveRanges map[*ast.VariableDefinition]LiveRange

	// Where data are located immediately prior to executing the block.
	// Every entry maps one-to-one to the corresponding live in set.
	LocationIn LocationSet

	// Where data are located immediately after the block executed.
	// Every entry maps one-to-one to the corresponding live in set.
	LocationOut LocationSet
}

func (state *BlockState) ComputeLiveRanges() {
	defStarts := map[*ast.VariableDefinition]int{}
	for def, _ := range state.LiveIn {
		defStarts[def] = 0
	}

	liveRanges := map[*ast.VariableDefinition]LiveRange{}
	for idx, inst := range state.Instructions {
		dist := idx + 1

		def := inst.Destination()
		if def != nil {
			defStarts[def] = dist
		}

		for _, src := range inst.Sources() {
			ref, ok := src.(*ast.VariableReference)
			if !ok {
				continue
			}

			liveRanges[ref.UseDef] = LiveRange{
				Start: defStarts[ref.UseDef],
				End:   dist,
			}
		}
	}

	for def, info := range state.LiveOut {
		liveRanges[def] = LiveRange{
			Start: defStarts[def],
			End:   info.Distance,
		}
	}

	state.LiveRanges = liveRanges
}
