package util

import ghutil "gohopper/core/util"

// EdgeFilter decides whether an edge should be accepted during graph traversal.
type EdgeFilter func(edgeState ghutil.EdgeIteratorState) bool

// AllEdges accepts every edge.
var AllEdges EdgeFilter = func(_ ghutil.EdgeIteratorState) bool { return true }
