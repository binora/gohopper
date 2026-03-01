package util

import ghutil "gohopper/core/util"

// TraversalMode determines how the graph is traversed, either node-based or edge-based.
type TraversalMode int

const (
	// NodeBased traversal identifies SPT entries by adjacent node ID.
	NodeBased TraversalMode = iota
	// EdgeBased traversal identifies SPT entries by edge key.
	EdgeBased
)

// IsEdgeBased reports whether this mode is edge-based.
func (m TraversalMode) IsEdgeBased() bool {
	return m == EdgeBased
}

// CreateTraversalID returns the identifier to access the shortest-path-tree map.
// For NodeBased mode it returns the adjacent node ID.
// For EdgeBased mode it returns the edge key (or the reverse edge key if reverse is true).
func (m TraversalMode) CreateTraversalID(edgeState ghutil.EdgeIteratorState, reverse bool) int {
	if m.IsEdgeBased() {
		if reverse {
			return edgeState.GetReverseEdgeKey()
		}
		return edgeState.GetEdgeKey()
	}
	return edgeState.GetAdjNode()
}

// String returns a human-readable name for the traversal mode.
func (m TraversalMode) String() string {
	switch m {
	case NodeBased:
		return "NODE_BASED"
	case EdgeBased:
		return "EDGE_BASED"
	default:
		return "UNKNOWN"
	}
}
