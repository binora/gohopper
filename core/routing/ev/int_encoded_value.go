package ev

// IntEncodedValue defines storage and retrieval of unsigned integer edge
// properties. The range is limited to fit within the allocated bits (max 32).
type IntEncodedValue interface {
	EncodedValue
	GetInt(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess) int32
	SetInt(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess, value int32)
	GetMaxStorableInt() int32
	GetMinStorableInt() int32
	GetMaxOrMaxStorableInt() int32
}
