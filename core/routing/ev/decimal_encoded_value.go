package ev

// DecimalEncodedValue defines storage and retrieval of unsigned decimal
// edge properties. The range is limited by the number of allocated bits
// and a scaling factor. The default value is always 0.
type DecimalEncodedValue interface {
	EncodedValue

	// SetDecimal stores the specified float64 value (rounded with a
	// previously defined factor) into the edge storage.
	SetDecimal(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess, value float64)

	// GetDecimal retrieves the decimal value from the edge storage.
	GetDecimal(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess) float64

	// GetMaxStorableDecimal returns the maximum value accepted by SetDecimal.
	GetMaxStorableDecimal() float64

	// GetMinStorableDecimal returns the minimum value accepted by SetDecimal.
	GetMinStorableDecimal() float64

	// GetMaxOrMaxStorableDecimal returns the maximum value that has been set,
	// or the physical storage limit if no value has been set yet.
	GetMaxOrMaxStorableDecimal() float64

	// GetNextStorableValue returns the smallest decimal value that is >= the
	// given value and can be stored exactly (i.e. survives a set/get round-trip).
	GetNextStorableValue(value float64) float64

	// GetSmallestNonZeroValue returns the smallest positive value that can
	// be represented.
	GetSmallestNonZeroValue() float64
}
