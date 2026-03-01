package util

import ghutil "gohopper/core/util"

// DirectedEdgeFilter decides whether an edge should be accepted during
// graph traversal, taking the traversal direction into account.
type DirectedEdgeFilter func(edgeState ghutil.EdgeIteratorState, reverse bool) bool
