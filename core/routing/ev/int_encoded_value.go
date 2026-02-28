package ev

// IntEncodedValue defines storage and retrieval of unsigned integer edge
// properties. The range is limited to fit within the allocated bits
// (maximum 32). The default value is always 0.
type IntEncodedValue interface {
	EncodedValue

	// GetInt retrieves the integer value from the edge storage.
	GetInt(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess) int32

	// SetInt stores the integer value into the edge storage.
	SetInt(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess, value int32)

	// GetMaxStorableInt returns the maximum value accepted by SetInt.
	GetMaxStorableInt() int32

	// GetMinStorableInt returns the minimum value accepted by SetInt.
	GetMinStorableInt() int32

	// GetMaxOrMaxStorableInt returns the maximum value that has been set,
	// or the physical storage limit if no value has been set yet.
	GetMaxOrMaxStorableInt() int32
}
