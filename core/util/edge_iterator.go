package util

const (
	// NoEdge is used where an edge would normally be expected but none exists
	// (e.g. the root of the shortest-path tree has no parent edge).
	NoEdge = -1

	// AnyEdge is used where an edge is expected but no specific edge is required.
	AnyEdge = -2
)

// EdgeIterator iterates through edges of one node. It extends
// EdgeIteratorState with a Next() method for iteration.
type EdgeIterator interface {
	EdgeIteratorState

	// Next advances to the next edge. Returns true if an edge is available.
	Next() bool
}

// EdgeIsValid returns true if the edge id is >= 0.
func EdgeIsValid(edgeID int) bool {
	return edgeID >= 0
}
