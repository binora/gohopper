package routing

// PathCalculator calculates paths for different start/target nodes and edge restrictions.
type PathCalculator interface {
	CalcPaths(from, to int, restrictions EdgeRestrictions) []*Path
	GetDebugString() string
	GetVisitedNodes() int
}
