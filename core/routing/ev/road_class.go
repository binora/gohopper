package ev

import (
	"fmt"
	"strings"
)

// Compile-time interface compliance check.
var _ fmt.Stringer = RoadClass(0)

// RoadClass defines the road class of an edge, heavily influenced by the
// highway tag in OSM. All edges that do not fit get RoadClassOther.
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
)

// RoadClassKey is the encoded value key for road class.
const RoadClassKey = "road_class"

// roadClassValues holds all RoadClass constants in ordinal order.
var roadClassValues = []RoadClass{
	RoadClassOther, RoadClassMotorway, RoadClassTrunk, RoadClassPrimary,
	RoadClassSecondary, RoadClassTertiary, RoadClassResidential, RoadClassUnclassified,
	RoadClassService, RoadClassRoad, RoadClassTrack, RoadClassBridleway,
	RoadClassSteps, RoadClassCycleway, RoadClassPath, RoadClassLivingStreet,
	RoadClassFootway, RoadClassPedestrian, RoadClassPlatform, RoadClassCorridor,
	RoadClassConstruction, RoadClassBusway,
}

// roadClassNames maps each RoadClass to its lowercase string representation.
var roadClassNames = [...]string{
	"other", "motorway", "trunk", "primary",
	"secondary", "tertiary", "residential", "unclassified",
	"service", "road", "track", "bridleway",
	"steps", "cycleway", "path", "living_street",
	"footway", "pedestrian", "platform", "corridor",
	"construction", "busway",
}

// String returns the lowercase representation of the road class.
func (rc RoadClass) String() string {
	if rc >= 0 && int(rc) < len(roadClassNames) {
		return roadClassNames[rc]
	}
	return "other"
}

// RoadClassFind returns the RoadClass matching the given name, or
// RoadClassOther if not found.
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

// RoadClassCreate creates an EnumEncodedValue for RoadClass.
func RoadClassCreate() *EnumEncodedValue[RoadClass] {
	return NewEnumEncodedValue[RoadClass](RoadClassKey, roadClassValues)
}
