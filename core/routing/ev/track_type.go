package ev

import (
	"fmt"
	"strings"
)

// Compile-time interface compliance check.
var _ fmt.Stringer = TrackType(0)

// TrackType defines the track type of an edge which describes how
// well-maintained a certain track is. Grade1 is very well-maintained,
// Grade5 is poorly maintained.
type TrackType int

const (
	TrackTypeMissing TrackType = iota
	TrackTypeGrade1
	TrackTypeGrade2
	TrackTypeGrade3
	TrackTypeGrade4
	TrackTypeGrade5
)

// TrackTypeKey is the encoded value key for track type.
const TrackTypeKey = "track_type"

// trackTypeValues holds all TrackType constants in ordinal order.
var trackTypeValues = []TrackType{
	TrackTypeMissing, TrackTypeGrade1, TrackTypeGrade2,
	TrackTypeGrade3, TrackTypeGrade4, TrackTypeGrade5,
}

// trackTypeNames maps each TrackType to its lowercase string representation.
var trackTypeNames = [...]string{
	"missing", "grade1", "grade2", "grade3", "grade4", "grade5",
}

// String returns the lowercase representation of the track type.
func (t TrackType) String() string {
	if t >= 0 && int(t) < len(trackTypeNames) {
		return trackTypeNames[t]
	}
	return "missing"
}

// TrackTypeFind returns the TrackType matching the given name, or
// TrackTypeMissing if not found.
func TrackTypeFind(name string) TrackType {
	if name == "" {
		return TrackTypeMissing
	}
	for i, n := range trackTypeNames {
		if strings.EqualFold(n, name) {
			return TrackType(i)
		}
	}
	return TrackTypeMissing
}

// TrackTypeCreate creates an EnumEncodedValue for TrackType.
func TrackTypeCreate() *EnumEncodedValue[TrackType] {
	return NewEnumEncodedValue[TrackType](TrackTypeKey, trackTypeValues)
}
