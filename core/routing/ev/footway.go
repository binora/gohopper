package ev

import (
	"fmt"
	"strings"
)

var _ fmt.Stringer = Footway(0)

type Footway int

const (
	FootwayMissing Footway = iota
	FootwaySidewalk
	FootwayCrossing
	FootwayAccessAisle
	FootwayLink
	FootwayTrafficIsland
	FootwayAlley
	footwayCount
)

const FootwayKey = "footway"

var footwayNames = [...]string{
	"missing", "sidewalk", "crossing", "access_aisle",
	"link", "traffic_island", "alley",
}

func (f Footway) String() string {
	if f >= 0 && int(f) < len(footwayNames) {
		return footwayNames[f]
	}
	return "missing"
}

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

func FootwayCreate() *EnumEncodedValue[Footway] {
	return NewEnumEncodedValue(FootwayKey, enumSequence[Footway](int(footwayCount)))
}
