package ev

// OrientationKey is the encoded value key for edge orientation (bearing).
const OrientationKey = "orientation"

// OrientationCreate creates a DecimalEncodedValue for edge orientation
// in degrees.
func OrientationCreate() DecimalEncodedValue {
	return NewDecimalEncodedValueImplFull(OrientationKey, 5, 0, 360.0/30.0, false, true, false)
}
