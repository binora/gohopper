package ev

import (
	"fmt"
	"strings"
)

// Compile-time interface compliance check.
var _ fmt.Stringer = Smoothness(0)

// Smoothness defines the road smoothness of an edge. If not tagged the value
// will be SmoothnessMissing. All unknown smoothness tags get SmoothnessOther.
type Smoothness int

const (
	SmoothnessMissing Smoothness = iota
	SmoothnessExcellent
	SmoothnessGood
	SmoothnessIntermediate
	SmoothnessBad
	SmoothnessVeryBad
	SmoothnessHorrible
	SmoothnessVeryHorrible
	SmoothnessImpassable
	SmoothnessOther
)

// SmoothnessKey is the encoded value key for smoothness.
const SmoothnessKey = "smoothness"

// smoothnessValues holds all Smoothness constants in ordinal order.
var smoothnessValues = []Smoothness{
	SmoothnessMissing, SmoothnessExcellent, SmoothnessGood, SmoothnessIntermediate,
	SmoothnessBad, SmoothnessVeryBad, SmoothnessHorrible, SmoothnessVeryHorrible,
	SmoothnessImpassable, SmoothnessOther,
}

// smoothnessNames maps each Smoothness to its lowercase string representation.
var smoothnessNames = [...]string{
	"missing", "excellent", "good", "intermediate",
	"bad", "very_bad", "horrible", "very_horrible",
	"impassable", "other",
}

// String returns the lowercase representation of the smoothness.
func (s Smoothness) String() string {
	if s >= 0 && int(s) < len(smoothnessNames) {
		return smoothnessNames[s]
	}
	return "missing"
}

// SmoothnessFind returns the Smoothness matching the given name, or
// SmoothnessOther if not found.
func SmoothnessFind(name string) Smoothness {
	if name == "" {
		return SmoothnessMissing
	}
	for i, n := range smoothnessNames {
		if strings.EqualFold(n, name) {
			return Smoothness(i)
		}
	}
	return SmoothnessOther
}

// SmoothnessCreate creates an EnumEncodedValue for Smoothness.
func SmoothnessCreate() *EnumEncodedValue[Smoothness] {
	return NewEnumEncodedValue[Smoothness](SmoothnessKey, smoothnessValues)
}
