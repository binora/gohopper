package ev

// MaxHeightKey is the encoded value key for maximum height.
const MaxHeightKey = "max_height"

// MaxHeightCreate creates a DecimalEncodedValue for maximum height (metres).
func MaxHeightCreate() DecimalEncodedValue {
	return NewDecimalEncodedValueImplFull(MaxHeightKey, 7, 0, 0.1, false, false, true)
}
