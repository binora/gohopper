package ev

// MaxSlopeKey is the encoded value key for maximum slope.
const MaxSlopeKey = "max_slope"

// MaxSlopeCreate creates a DecimalEncodedValue for maximum slope (degrees).
func MaxSlopeCreate() DecimalEncodedValue {
	return NewDecimalEncodedValueImplFull(MaxSlopeKey, 5, 0, 1, true, false, false)
}
