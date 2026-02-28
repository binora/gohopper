package ev

import (
	"fmt"
	"strings"
)

// Compile-time interface compliance check.
var _ fmt.Stringer = Crossing(0)

// Crossing defines the crossing type of an edge.
type Crossing int

const (
	CrossingMissing Crossing = iota
	CrossingRailwayBarrier
	CrossingRailway
	CrossingTrafficSignals
	CrossingUncontrolled
	CrossingMarked
	CrossingUnmarked
	CrossingNo
)

// CrossingKey is the encoded value key for crossing.
const CrossingKey = "crossing"

// crossingValues holds all Crossing constants in ordinal order.
var crossingValues = []Crossing{
	CrossingMissing, CrossingRailwayBarrier, CrossingRailway,
	CrossingTrafficSignals, CrossingUncontrolled, CrossingMarked,
	CrossingUnmarked, CrossingNo,
}

// crossingNames maps each Crossing to its lowercase string representation.
var crossingNames = [...]string{
	"missing", "railway_barrier", "railway", "traffic_signals",
	"uncontrolled", "marked", "unmarked", "no",
}

// String returns the lowercase representation of the crossing type.
func (c Crossing) String() string {
	if c >= 0 && int(c) < len(crossingNames) {
		return crossingNames[c]
	}
	return "missing"
}

// CrossingFind returns the Crossing matching the given name, or
// CrossingMissing if not found.
func CrossingFind(name string) Crossing {
	if name == "" {
		return CrossingMissing
	}
	for i, n := range crossingNames {
		if strings.EqualFold(n, name) {
			return Crossing(i)
		}
	}
	return CrossingMissing
}

// CrossingCreate creates an EnumEncodedValue for Crossing.
func CrossingCreate() *EnumEncodedValue[Crossing] {
	return NewEnumEncodedValue[Crossing](CrossingKey, crossingValues)
}
