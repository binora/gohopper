package ev

// MaxWidthKey is the encoded value key for maximum width.
const MaxWidthKey = "max_width"

// MaxWidthCreate creates a DecimalEncodedValue for maximum width (metres).
func MaxWidthCreate() DecimalEncodedValue {
	return NewDecimalEncodedValueImplFull(MaxWidthKey, 7, 0, 0.1, false, false, true)
}
