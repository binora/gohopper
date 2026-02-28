package ev

// EdgeIntAccess provides indexed int32 access for edge properties.
type EdgeIntAccess interface {
	GetInt(edgeID, index int) int32
	SetInt(edgeID, index int, value int32)
}
