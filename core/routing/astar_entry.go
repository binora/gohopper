package routing

import "gohopper/core/util"

// NewAStarSPTEntry creates an SPTEntry for A* where the heap weight includes
// the heuristic but WeightOfVisitedPath stores the actual path weight.
func NewAStarSPTEntry(edgeID, adjNode int, heapWeight, pathWeight float64, parent *SPTEntry) *SPTEntry {
	return NewSPTEntryWithHeuristic(edgeID, adjNode, heapWeight, pathWeight, parent)
}

// NewAStarSPTEntryRoot creates a root SPTEntry for A* with no edge and no parent.
func NewAStarSPTEntryRoot(node int, heapWeight, pathWeight float64) *SPTEntry {
	return NewSPTEntryWithHeuristic(util.NoEdge, node, heapWeight, pathWeight, nil)
}
