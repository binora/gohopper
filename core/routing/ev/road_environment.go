package ev

import (
	"fmt"
	"strings"
)

var _ fmt.Stringer = RoadEnvironment(0)

type RoadEnvironment int

const (
	RoadEnvironmentOther RoadEnvironment = iota
	RoadEnvironmentRoad
	RoadEnvironmentFerry
	RoadEnvironmentTunnel
	RoadEnvironmentBridge
	RoadEnvironmentFord
	roadEnvironmentCount
)

const RoadEnvironmentKey = "road_environment"

var roadEnvironmentNames = [...]string{
	"other", "road", "ferry", "tunnel", "bridge", "ford",
}

func (r RoadEnvironment) String() string {
	if r >= 0 && int(r) < len(roadEnvironmentNames) {
		return roadEnvironmentNames[r]
	}
	return "other"
}

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

func RoadEnvironmentCreate() *EnumEncodedValue[RoadEnvironment] {
	return NewEnumEncodedValue[RoadEnvironment](RoadEnvironmentKey, enumSequence[RoadEnvironment](int(roadEnvironmentCount)))
}
