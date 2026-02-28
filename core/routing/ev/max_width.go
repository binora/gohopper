package ev

const MaxWidthKey = "max_width"

func MaxWidthCreate() DecimalEncodedValue {
	return NewDecimalEncodedValueImplFull(MaxWidthKey, 7, 0, 0.1, false, false, true)
}
