package ev

// LanesKey is the encoded value key for the number of lanes.
const LanesKey = "lanes"

// LanesCreate creates an IntEncodedValue for the number of lanes.
func LanesCreate() IntEncodedValue {
	return NewIntEncodedValueImpl(LanesKey, 3, false)
}
