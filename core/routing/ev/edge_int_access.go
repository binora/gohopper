package ev

// EdgeIntAccess provides indexed integer access for edge properties.
// Each edge has a fixed number of int32 slots; callers address them by
// (edgeID, index) pairs.
type EdgeIntAccess interface {
	GetInt(edgeID, index int) int32
	SetInt(edgeID, index int, value int32)
}
