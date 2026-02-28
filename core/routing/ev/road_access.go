package ev

import (
	"fmt"
	"strings"
)

// Compile-time interface compliance check.
var _ fmt.Stringer = RoadAccess(0)

// RoadAccess defines the road access of an edge. Most edges are accessible
// from everyone and so the default value is RoadAccessYes.
type RoadAccess int

const (
	RoadAccessYes RoadAccess = iota
	RoadAccessDestination
	RoadAccessCustomers
	RoadAccessDelivery
	RoadAccessPrivate
	RoadAccessMilitary
	RoadAccessAgricultural
	RoadAccessForestry
	RoadAccessNo
)

// RoadAccessKey is the encoded value key for road access.
const RoadAccessKey = "road_access"

// roadAccessValues holds all RoadAccess constants in ordinal order.
var roadAccessValues = []RoadAccess{
	RoadAccessYes, RoadAccessDestination, RoadAccessCustomers, RoadAccessDelivery,
	RoadAccessPrivate, RoadAccessMilitary, RoadAccessAgricultural, RoadAccessForestry,
	RoadAccessNo,
}

// roadAccessNames maps each RoadAccess to its lowercase string representation.
var roadAccessNames = [...]string{
	"yes", "destination", "customers", "delivery",
	"private", "military", "agricultural", "forestry", "no",
}

// String returns the lowercase representation of the road access.
func (r RoadAccess) String() string {
	if r >= 0 && int(r) < len(roadAccessNames) {
		return roadAccessNames[r]
	}
	return "yes"
}

// RoadAccessFind returns the RoadAccess matching the given name, or
// RoadAccessYes if not found.
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

// RoadAccessCreate creates an EnumEncodedValue for RoadAccess.
func RoadAccessCreate() *EnumEncodedValue[RoadAccess] {
	return NewEnumEncodedValue[RoadAccess](RoadAccessKey, roadAccessValues)
}
