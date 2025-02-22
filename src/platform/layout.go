package platform

type RelocationKind string

const (
	// x64-style 32-bit relative offset where the offset is relative to the
	// end of the offset bytes / next instruction (EIP).  i.e.,
	//
	// Rel32Relocation = int32(
	//   LabelledEntryLocation - (CurrentEntryLocation + LocationOffset + 4))
	//
	// NOTE: This is equivalent to SystemV ABI's R_X86_64_PLT32 with A = 4.
	Rel32Relocation = RelocationKind("rel32")

	// Labelled entry's absolute 64-bit location.
	//
	// NOTE: This is equivalent to SystemV ABI's R_X86_64_64 with A = 0.
	Abs64Relocation = RelocationKind("abs64")
)

type SegmentLabel struct {
	Name    string
	IsLocal bool // true for block label
}

type Relocation struct {
	Kind RelocationKind

	Offset int // relative to the beginning of the segment

	Label SegmentLabel
}

// A continuous segment of instruction bytes.
type Segment struct {
	Bytes       []byte
	Relocations []Relocation
}
