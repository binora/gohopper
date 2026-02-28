package util

import ghutil "gohopper/core/util"

// EdgeFilter decides whether an edge should be accepted during graph
// traversal. It mirrors Java's com.graphhopper.routing.util.EdgeFilter.
type EdgeFilter func(edgeState ghutil.EdgeIteratorState) bool

// AllEdges is an EdgeFilter that accepts every edge.
var AllEdges EdgeFilter = func(_ ghutil.EdgeIteratorState) bool { return true }
