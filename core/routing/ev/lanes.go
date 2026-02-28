package ev

const LanesKey = "lanes"

func LanesCreate() IntEncodedValue {
	return NewIntEncodedValueImpl(LanesKey, 3, false)
}
