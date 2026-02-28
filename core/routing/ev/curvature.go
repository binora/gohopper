package ev

// CurvatureKey is the encoded value key for edge curvature.
const CurvatureKey = "curvature"

// CurvatureCreate creates a DecimalEncodedValue for edge curvature,
// representing the ratio of beeline distance to actual distance.
func CurvatureCreate() DecimalEncodedValue {
	return NewDecimalEncodedValueImplFull(CurvatureKey, 4, 0.25, 0.05, false, false, false)
}
