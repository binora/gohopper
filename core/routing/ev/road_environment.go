package ev

import (
	"fmt"
	"strings"
)

// Compile-time interface compliance check.
var _ fmt.Stringer = RoadEnvironment(0)

// RoadEnvironment defines the road environment of an edge.
type RoadEnvironment int

const (
	RoadEnvironmentOther RoadEnvironment = iota
	RoadEnvironmentRoad
	RoadEnvironmentFerry
	RoadEnvironmentTunnel
	RoadEnvironmentBridge
	RoadEnvironmentFord
)

// RoadEnvironmentKey is the encoded value key for road environment.
const RoadEnvironmentKey = "road_environment"

// roadEnvironmentValues holds all RoadEnvironment constants in ordinal order.
var roadEnvironmentValues = []RoadEnvironment{
	RoadEnvironmentOther, RoadEnvironmentRoad, RoadEnvironmentFerry,
	RoadEnvironmentTunnel, RoadEnvironmentBridge, RoadEnvironmentFord,
}

// roadEnvironmentNames maps each RoadEnvironment to its lowercase string representation.
var roadEnvironmentNames = [...]string{
	"other", "road", "ferry", "tunnel", "bridge", "ford",
}

// String returns the lowercase representation of the road environment.
func (r RoadEnvironment) String() string {
	if r >= 0 && int(r) < len(roadEnvironmentNames) {
		return roadEnvironmentNames[r]
	}
	return "other"
}

// RoadEnvironmentFind returns the RoadEnvironment matching the given name, or
// RoadEnvironmentOther if not found.
func RoadEnvironmentFind(name string) RoadEnvironment {
	if name == "" {
		return RoadEnvironmentOther
	}
	for i, n := range roadEnvironmentNames {
		if strings.EqualFold(n, name) {
			return RoadEnvironment(i)
		}
	}
	return RoadEnvironmentOther
}

// RoadEnvironmentCreate creates an EnumEncodedValue for RoadEnvironment.
func RoadEnvironmentCreate() *EnumEncodedValue[RoadEnvironment] {
	return NewEnumEncodedValue[RoadEnvironment](RoadEnvironmentKey, roadEnvironmentValues)
}
