package ev

import (
	"fmt"
	"strings"
)

var _ fmt.Stringer = TrackType(0)

// TrackType defines how well-maintained a track is.
// Grade1 is very well-maintained, Grade5 is poorly maintained.
type TrackType int

const (
	TrackTypeMissing TrackType = iota
	TrackTypeGrade1
	TrackTypeGrade2
	TrackTypeGrade3
	TrackTypeGrade4
	TrackTypeGrade5
	trackTypeCount
)

const TrackTypeKey = "track_type"

var trackTypeNames = [...]string{
	"missing", "grade1", "grade2", "grade3", "grade4", "grade5",
}

func (t TrackType) String() string {
	if t >= 0 && int(t) < len(trackTypeNames) {
		return trackTypeNames[t]
	}
	return "missing"
}

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

func TrackTypeCreate() *EnumEncodedValue[TrackType] {
	return NewEnumEncodedValue(TrackTypeKey, enumSequence[TrackType](int(trackTypeCount)))
}
