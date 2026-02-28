package ev

const MaxSlopeKey = "max_slope"

func MaxSlopeCreate() DecimalEncodedValue {
	return NewDecimalEncodedValueImplFull(MaxSlopeKey, 5, 0, 1, true, false, false)
}
