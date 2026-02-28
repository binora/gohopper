package ev

const MaxAxleLoadKey = "max_axle_load"

func MaxAxleLoadCreate() DecimalEncodedValue {
	return NewDecimalEncodedValueImplFull(MaxAxleLoadKey, 7, 0, 0.5, false, false, true)
}
