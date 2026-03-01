package routing

import (
	"fmt"

	"gohopper/core/util"
)

// SPTEntry represents a node in the shortest-path tree. For A* algorithms,
// Weight includes the heuristic estimate while WeightOfVisitedPath stores the
// actual path weight.
type SPTEntry struct {
	Edge                int
	AdjNode             int
	Weight              float64
	WeightOfVisitedPath float64
	Parent              *SPTEntry
	Deleted             bool
}

func NewSPTEntry(node int, weight float64) *SPTEntry {
	return &SPTEntry{
		Edge:                util.NoEdge,
		AdjNode:             node,
		Weight:              weight,
		WeightOfVisitedPath: weight,
	}
}

func NewSPTEntryFull(edgeID, adjNode int, weight float64, parent *SPTEntry) *SPTEntry {
	return &SPTEntry{
		Edge:                edgeID,
		AdjNode:             adjNode,
		Weight:              weight,
		WeightOfVisitedPath: weight,
		Parent:              parent,
	}
}

// NewSPTEntryWithHeuristic creates an SPTEntry where the heap weight differs
// from the actual path weight (used by A* algorithms).
func NewSPTEntryWithHeuristic(edgeID, adjNode int, heapWeight, pathWeight float64, parent *SPTEntry) *SPTEntry {
	return &SPTEntry{
		Edge:                edgeID,
		AdjNode:             adjNode,
		Weight:              heapWeight,
		WeightOfVisitedPath: pathWeight,
		Parent:              parent,
	}
}

func (e *SPTEntry) GetWeightOfVisitedPath() float64 {
	return e.WeightOfVisitedPath
}

func (e *SPTEntry) Less(other *SPTEntry) bool {
	return e.Weight < other.Weight
}

func (e *SPTEntry) String() string {
	return fmt.Sprintf("%d (%d) weight: %v", e.AdjNode, e.Edge, e.Weight)
}
