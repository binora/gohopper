package ev

const MaxWeightKey = "max_weight"

func MaxWeightCreate() DecimalEncodedValue {
	return NewDecimalEncodedValueImplFull(MaxWeightKey, 9, 0, 0.1, false, false, true)
}
