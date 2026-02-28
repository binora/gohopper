package ev

import (
	"fmt"
	"strings"
)

// Compile-time interface compliance check.
var _ fmt.Stringer = Footway(0)

// Footway defines the footway type of an edge.
type Footway int

const (
	FootwayMissing Footway = iota
	FootwaySidewalk
	FootwayCrossing
	FootwayAccessAisle
	FootwayLink
	FootwayTrafficIsland
	FootwayAlley
)

// FootwayKey is the encoded value key for footway.
const FootwayKey = "footway"

// footwayValues holds all Footway constants in ordinal order.
var footwayValues = []Footway{
	FootwayMissing, FootwaySidewalk, FootwayCrossing, FootwayAccessAisle,
	FootwayLink, FootwayTrafficIsland, FootwayAlley,
}

// footwayNames maps each Footway to its lowercase string representation.
var footwayNames = [...]string{
	"missing", "sidewalk", "crossing", "access_aisle",
	"link", "traffic_island", "alley",
}

// String returns the lowercase representation of the footway type.
func (f Footway) String() string {
	if f >= 0 && int(f) < len(footwayNames) {
		return footwayNames[f]
	}
	return "missing"
}

// FootwayFind returns the Footway matching the given name, or
// FootwayMissing if not found.
func FootwayFind(name string) Footway {
	if name == "" {
		return FootwayMissing
	}
	for i, n := range footwayNames {
		if strings.EqualFold(n, name) {
			return Footway(i)
		}
	}
	return FootwayMissing
}

// FootwayCreate creates an EnumEncodedValue for Footway.
func FootwayCreate() *EnumEncodedValue[Footway] {
	return NewEnumEncodedValue[Footway](FootwayKey, footwayValues)
}
