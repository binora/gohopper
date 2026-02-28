package ev

import (
	"fmt"
	"strings"
)

var _ fmt.Stringer = RoadAccess(0)

type RoadAccess int

const (
	RoadAccessYes RoadAccess = iota
	RoadAccessDestination
	RoadAccessCustomers
	RoadAccessDelivery
	RoadAccessPrivate
	RoadAccessAgricultural
	RoadAccessForestry
	RoadAccessNo
	roadAccessCount
)

const RoadAccessKey = "road_access"

var roadAccessNames = [...]string{
	"yes", "destination", "customers", "delivery",
	"private", "agricultural", "forestry", "no",
}

func (r RoadAccess) String() string {
	if r >= 0 && int(r) < len(roadAccessNames) {
		return roadAccessNames[r]
	}
	return "yes"
}

// RoadAccessFind maps a name to a RoadAccess value.
// "permit" and "service" are treated as RoadAccessPrivate.
func RoadAccessFind(name string) RoadAccess {
	if name == "" {
		return RoadAccessYes
	}
	if strings.EqualFold(name, "permit") || strings.EqualFold(name, "service") {
		return RoadAccessPrivate
	}
	for i, n := range roadAccessNames {
		if strings.EqualFold(n, name) {
			return RoadAccess(i)
		}
	}
	return RoadAccessYes
}

func RoadAccessCreate() *EnumEncodedValue[RoadAccess] {
	return NewEnumEncodedValue(RoadAccessKey, enumSequence[RoadAccess](int(roadAccessCount)))
}
