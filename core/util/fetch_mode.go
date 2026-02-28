package util

// FetchMode specifies which nodes to include in the PointList returned by
// EdgeIteratorState.FetchWayGeometry.
type FetchMode int

const (
	FetchModeTowerOnly    FetchMode = iota
	FetchModePillarOnly
	FetchModeBaseAndPillar
	FetchModePillarAndAdj
	FetchModeAll
)

var fetchModeNames = [...]string{
	"TOWER_ONLY", "PILLAR_ONLY", "BASE_AND_PILLAR", "PILLAR_AND_ADJ", "ALL",
}

func (m FetchMode) String() string {
	if m >= 0 && int(m) < len(fetchModeNames) {
		return fetchModeNames[m]
	}
	return "UNKNOWN"
}
