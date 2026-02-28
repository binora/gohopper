package ev

import (
	"fmt"
	"strings"
)

var _ fmt.Stringer = Smoothness(0)

// Smoothness defines the road smoothness of an edge.
// SmoothnessMissing for untagged, SmoothnessOther for unrecognized tags.
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
	smoothnessCount
)

const SmoothnessKey = "smoothness"

var smoothnessNames = [...]string{
	"missing", "excellent", "good", "intermediate",
	"bad", "very_bad", "horrible", "very_horrible",
	"impassable", "other",
}

func (s Smoothness) String() string {
	if s >= 0 && int(s) < len(smoothnessNames) {
		return smoothnessNames[s]
	}
	return "missing"
}

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

func SmoothnessCreate() *EnumEncodedValue[Smoothness] {
	return NewEnumEncodedValue[Smoothness](SmoothnessKey, enumSequence[Smoothness](int(smoothnessCount)))
}
