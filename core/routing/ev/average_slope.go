package ev

const AverageSlopeKey = "average_slope"

func AverageSlopeCreate() DecimalEncodedValue {
	return NewDecimalEncodedValueImplFull(AverageSlopeKey, 5, 0, 1, true, false, false)
}
