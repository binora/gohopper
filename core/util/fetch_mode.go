package util

// FetchMode specifies which nodes to include in the PointList returned by
// EdgeIteratorState.FetchWayGeometry.
type FetchMode int

const (
	FetchModeTowerOnly    FetchMode = iota // only tower (junction) nodes
	FetchModePillarOnly                    // only pillar (intermediate) nodes
	FetchModeBaseAndPillar                 // base node + pillar nodes
	FetchModePillarAndAdj                  // pillar nodes + adjacent node
	FetchModeAll                           // base + pillar + adjacent
)

func (m FetchMode) String() string {
	switch m {
	case FetchModeTowerOnly:
		return "TOWER_ONLY"
	case FetchModePillarOnly:
		return "PILLAR_ONLY"
	case FetchModeBaseAndPillar:
		return "BASE_AND_PILLAR"
	case FetchModePillarAndAdj:
		return "PILLAR_AND_ADJ"
	case FetchModeAll:
		return "ALL"
	default:
		return "UNKNOWN"
	}
}
