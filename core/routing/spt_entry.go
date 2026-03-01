package routing

import (
	"fmt"

	"gohopper/core/util"
)

// SPTEntry represents a node in the shortest-path-tree built from linked entries.
// For A* algorithms, Weight includes the heuristic estimate (used for heap ordering)
// while WeightOfVisitedPath stores the actual path weight without heuristic.
type SPTEntry struct {
	Edge                int       // edge ID, default NoEdge (-1)
	AdjNode             int       // adjacent node ID
	Weight              float64   // heap weight (for A*: path weight + heuristic)
	WeightOfVisitedPath float64   // actual path weight (for Dijkstra: same as Weight)
	Parent              *SPTEntry // parent entry in SPT
	Deleted             bool      // soft-delete flag for priority queue
}

// NewSPTEntry creates an SPTEntry with Edge set to NoEdge and no parent.
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

// NewSPTEntryWithHeuristic creates an SPTEntry where the heap weight differs from
// the actual path weight (used by A* algorithms).
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

// Less reports whether e has a smaller weight than other, suitable for min-heap ordering.
func (e *SPTEntry) Less(other *SPTEntry) bool {
	return e.Weight < other.Weight
}

// String returns a debug representation matching GraphHopper's format.
func (e *SPTEntry) String() string {
	return fmt.Sprintf("%d (%d) weight: %v", e.AdjNode, e.Edge, e.Weight)
}
