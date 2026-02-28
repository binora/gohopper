package ev

const FerrySpeedKey = "ferry_speed"

func FerrySpeedCreate() DecimalEncodedValue {
	return NewDecimalEncodedValueImpl(FerrySpeedKey, 5, 2, false)
}
