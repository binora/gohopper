package ev

// FerrySpeedKey is the encoded value key for ferry speed.
const FerrySpeedKey = "ferry_speed"

// FerrySpeedCreate creates a DecimalEncodedValue for ferry speed (km/h).
func FerrySpeedCreate() DecimalEncodedValue {
	return NewDecimalEncodedValueImpl(FerrySpeedKey, 5, 2, false)
}
