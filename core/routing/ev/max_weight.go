package ev

// MaxWeightKey is the encoded value key for maximum weight.
const MaxWeightKey = "max_weight"

// MaxWeightCreate creates a DecimalEncodedValue for maximum weight (tonnes).
func MaxWeightCreate() DecimalEncodedValue {
	return NewDecimalEncodedValueImplFull(MaxWeightKey, 9, 0, 0.1, false, false, true)
}
