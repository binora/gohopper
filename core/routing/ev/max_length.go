package ev

// MaxLengthKey is the encoded value key for maximum length.
const MaxLengthKey = "max_length"

// MaxLengthCreate creates a DecimalEncodedValue for maximum length (metres).
func MaxLengthCreate() DecimalEncodedValue {
	return NewDecimalEncodedValueImplFull(MaxLengthKey, 7, 0, 0.1, false, false, true)
}
