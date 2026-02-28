package ev

// AverageSlopeKey is the encoded value key for average slope.
const AverageSlopeKey = "average_slope"

// AverageSlopeCreate creates a DecimalEncodedValue for average slope (degrees).
func AverageSlopeCreate() DecimalEncodedValue {
	return NewDecimalEncodedValueImplFull(AverageSlopeKey, 5, 0, 1, true, false, false)
}
