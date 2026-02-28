package ev

import (
	"fmt"
	"strings"
)

var _ fmt.Stringer = RoadClass(0)

// RoadClass defines the road class of an edge, heavily influenced by the
// highway tag in OSM.
type RoadClass int

const (
	RoadClassOther RoadClass = iota
	RoadClassMotorway
	RoadClassTrunk
	RoadClassPrimary
	RoadClassSecondary
	RoadClassTertiary
	RoadClassResidential
	RoadClassUnclassified
	RoadClassService
	RoadClassRoad
	RoadClassTrack
	RoadClassBridleway
	RoadClassSteps
	RoadClassCycleway
	RoadClassPath
	RoadClassLivingStreet
	RoadClassFootway
	RoadClassPedestrian
	RoadClassPlatform
	RoadClassCorridor
	RoadClassConstruction
	RoadClassBusway
	roadClassCount
)

const RoadClassKey = "road_class"

var roadClassNames = [...]string{
	"other", "motorway", "trunk", "primary",
	"secondary", "tertiary", "residential", "unclassified",
	"service", "road", "track", "bridleway",
	"steps", "cycleway", "path", "living_street",
	"footway", "pedestrian", "platform", "corridor",
	"construction", "busway",
}

func (rc RoadClass) String() string {
	if rc >= 0 && int(rc) < len(roadClassNames) {
		return roadClassNames[rc]
	}
	return "other"
}

func RoadClassFind(name string) RoadClass {
	if name == "" {
		return RoadClassOther
	}
	for i, n := range roadClassNames {
		if strings.EqualFold(n, name) {
			return RoadClass(i)
		}
	}
	return RoadClassOther
}

func RoadClassCreate() *EnumEncodedValue[RoadClass] {
	return NewEnumEncodedValue[RoadClass](RoadClassKey, enumSequence[RoadClass](int(roadClassCount)))
}
