package ev

const CurvatureKey = "curvature"

func CurvatureCreate() DecimalEncodedValue {
	return NewDecimalEncodedValueImplFull(CurvatureKey, 4, 0.25, 0.05, false, false, false)
}
