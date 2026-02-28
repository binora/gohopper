package ev

const MaxHeightKey = "max_height"

func MaxHeightCreate() DecimalEncodedValue {
	return NewDecimalEncodedValueImplFull(MaxHeightKey, 7, 0, 0.1, false, false, true)
}
