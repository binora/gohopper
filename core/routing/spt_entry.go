package routing

import (
	"fmt"

	"gohopper/core/util"
)

// SPTEntry represents a node in the shortest-path-tree built from linked entries.
type SPTEntry struct {
	Edge    int       // edge ID, default NoEdge (-1)
	AdjNode int       // adjacent node ID
	Weight  float64   // weight from start to this node
	Parent  *SPTEntry // parent entry in SPT
	Deleted bool      // soft-delete flag for priority queue
}

// NewSPTEntry creates an SPTEntry with Edge set to NoEdge and no parent.
func NewSPTEntry(node int, weight float64) *SPTEntry {
	return &SPTEntry{
		Edge:    util.NoEdge,
		AdjNode: node,
		Weight:  weight,
	}
}

// NewSPTEntryFull creates an SPTEntry with all fields specified.
func NewSPTEntryFull(edgeID, adjNode int, weight float64, parent *SPTEntry) *SPTEntry {
	return &SPTEntry{
		Edge:    edgeID,
		AdjNode: adjNode,
		Weight:  weight,
		Parent:  parent,
	}
}

// GetWeightOfVisitedPath returns the weight to the origin.
// Overridden in AStarEntry where Weight includes the heuristic estimate.
func (e *SPTEntry) GetWeightOfVisitedPath() float64 {
	return e.Weight
}

// Less reports whether e has a smaller weight than other, suitable for min-heap ordering.
func (e *SPTEntry) Less(other *SPTEntry) bool {
	return e.Weight < other.Weight
}

// String returns a debug representation matching GraphHopper's format.
func (e *SPTEntry) String() string {
	return fmt.Sprintf("%d (%d) weight: %v", e.AdjNode, e.Edge, e.Weight)
}
