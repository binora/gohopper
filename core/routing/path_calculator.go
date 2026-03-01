package routing

// PathCalculator allows repeatedly calculating paths for different start/target
// nodes and edge restrictions.
// Port of Java com.graphhopper.routing.PathCalculator.
type PathCalculator interface {
	// CalcPaths calculates one or more paths from 'from' to 'to' with the
	// given edge restrictions.
	CalcPaths(from, to int, restrictions EdgeRestrictions) []*Path

	// GetDebugString returns debug information from the last path calculation.
	GetDebugString() string

	// GetVisitedNodes returns the number of visited nodes from the last
	// path calculation.
	GetVisitedNodes() int
}
