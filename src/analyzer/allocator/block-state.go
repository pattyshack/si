package allocator

type BlockState struct {
	LiveIn  LiveSet
	LiveOut LiveSet

	// Where data are located immediately prior to executing the block.
	// Every entry maps one-to-one to the corresponding live in set.
	LocationIn LocationSet

	// Where data are located immediately after the block executed.
	// Every entry maps one-to-one to the corresponding live in set.
	LocationOut LocationSet
}
