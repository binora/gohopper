package ev

// EdgeIntAccess provides indexed integer access for edge properties.
// Each edge has a fixed number of int32 slots; callers address them by
// (edgeID, index) pairs.
type EdgeIntAccess interface {
	// GetInt returns the int32 value at the given slot index for the edge.
	GetInt(edgeID, index int) int32

	// SetInt stores the int32 value at the given slot index for the edge.
	SetInt(edgeID, index int, value int32)
}
