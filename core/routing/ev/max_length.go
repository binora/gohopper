package ev

const MaxLengthKey = "max_length"

func MaxLengthCreate() DecimalEncodedValue {
	return NewDecimalEncodedValueImplFull(MaxLengthKey, 7, 0, 0.1, false, false, true)
}
