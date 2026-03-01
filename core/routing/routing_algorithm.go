package routing

// RoutingAlgorithm calculates the shortest path from specified node IDs.
// An instance can be used only once.
type RoutingAlgorithm interface {
	// CalcPath calculates the best path between the specified nodes.
	// Call Found() on the returned Path to verify that the path is valid.
	CalcPath(from, to int) *Path

	// CalcPaths calculates multiple path possibilities.
	CalcPaths(from, to int) []*Path

	// SetMaxVisitedNodes limits the search to the given number of nodes.
	SetMaxVisitedNodes(numberOfNodes int)

	// SetTimeoutMillis limits the search to the given time in milliseconds.
	SetTimeoutMillis(timeoutMillis int64)

	// GetName returns the name of this algorithm.
	GetName() string

	// GetVisitedNodes returns the number of visited nodes after searching.
	// Useful for debugging.
	GetVisitedNodes() int
}
