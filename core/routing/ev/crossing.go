package ev

import (
	"fmt"
	"strings"
)

var _ fmt.Stringer = Crossing(0)

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
	crossingCount
)

const CrossingKey = "crossing"

var crossingNames = [...]string{
	"missing", "railway_barrier", "railway", "traffic_signals",
	"uncontrolled", "marked", "unmarked", "no",
}

func (c Crossing) String() string {
	if c >= 0 && int(c) < len(crossingNames) {
		return crossingNames[c]
	}
	return "missing"
}

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

func CrossingCreate() *EnumEncodedValue[Crossing] {
	return NewEnumEncodedValue[Crossing](CrossingKey, enumSequence[Crossing](int(crossingCount)))
}
