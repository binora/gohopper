package ev

const OrientationKey = "orientation"

func OrientationCreate() DecimalEncodedValue {
	return NewDecimalEncodedValueImplFull(OrientationKey, 5, 0, 360.0/30.0, false, true, false)
}
