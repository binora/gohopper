package ev

// DecimalEncodedValue defines storage and retrieval of decimal edge properties.
type DecimalEncodedValue interface {
	EncodedValue
	SetDecimal(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess, value float64)
	GetDecimal(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess) float64
	GetMaxStorableDecimal() float64
	GetMinStorableDecimal() float64
	GetMaxOrMaxStorableDecimal() float64
	GetNextStorableValue(value float64) float64
	GetSmallestNonZeroValue() float64
}
