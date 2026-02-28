package ev

// MaxAxleLoadKey is the encoded value key for maximum axle load.
const MaxAxleLoadKey = "max_axle_load"

// MaxAxleLoadCreate creates a DecimalEncodedValue for maximum axle load (tonnes).
func MaxAxleLoadCreate() DecimalEncodedValue {
	return NewDecimalEncodedValueImplFull(MaxAxleLoadKey, 7, 0, 0.5, false, false, true)
}
