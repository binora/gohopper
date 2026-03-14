package ch

import (
	"fmt"

	"gohopper/core/util"
)

// CHEntry extends SPTEntry with an incoming original edge field for CH routing.
type CHEntry struct {
	Edge    int
	IncEdge int
	AdjNode int
	Weight  float64
	Parent  *CHEntry
}

func NewCHEntry(node int, weight float64) *CHEntry {
	return &CHEntry{
		Edge:    util.NoEdge,
		IncEdge: util.NoEdge,
		AdjNode: node,
		Weight:  weight,
	}
}

func NewCHEntryFull(edge, incEdge, adjNode int, weight float64, parent *CHEntry) *CHEntry {
	return &CHEntry{
		Edge:    edge,
		IncEdge: incEdge,
		AdjNode: adjNode,
		Weight:  weight,
		Parent:  parent,
	}
}

func (e *CHEntry) String() string {
	return fmt.Sprintf("%d (%d) weight: %v, incEdge: %d", e.AdjNode, e.Edge, e.Weight, e.IncEdge)
}

// AStarCHEntry extends CHEntry with a separate visited-path weight for A* heuristic.
type AStarCHEntry struct {
	CHEntry
	WeightOfVisitedPath float64
	AStarParent         *AStarCHEntry
}

func NewAStarCHEntry(node int, heapWeight, weightOfVisitedPath float64) *AStarCHEntry {
	return &AStarCHEntry{
		CHEntry:             *NewCHEntry(node, heapWeight),
		WeightOfVisitedPath: weightOfVisitedPath,
	}
}

func NewAStarCHEntryFull(edge, incEdge, adjNode int, heapWeight, weightOfVisitedPath float64, parent *AStarCHEntry) *AStarCHEntry {
	return &AStarCHEntry{
		CHEntry:             *NewCHEntryFull(edge, incEdge, adjNode, heapWeight, &parent.CHEntry),
		WeightOfVisitedPath: weightOfVisitedPath,
		AStarParent:         parent,
	}
}

func (e *AStarCHEntry) GetParent() *AStarCHEntry {
	return e.AStarParent
}

func (e *AStarCHEntry) GetWeightOfVisitedPath() float64 {
	return e.WeightOfVisitedPath
}

// PrepareCHEntry is the priority queue entry used during CH preparation.
type PrepareCHEntry struct {
	IncEdgeKey  int
	FirstEdgeKey int
	OrigEdges   int
	PrepareEdge int
	AdjNode     int
	Weight      float64
	Parent      *PrepareCHEntry
}

func NewPrepareCHEntry(prepareEdge, firstEdgeKey, incEdgeKey, adjNode int, weight float64, origEdges int) *PrepareCHEntry {
	return &PrepareCHEntry{
		PrepareEdge:  prepareEdge,
		FirstEdgeKey: firstEdgeKey,
		IncEdgeKey:   incEdgeKey,
		AdjNode:      adjNode,
		Weight:       weight,
		OrigEdges:    origEdges,
	}
}

func (e *PrepareCHEntry) Less(other *PrepareCHEntry) bool {
	return e.Weight < other.Weight
}

func (e *PrepareCHEntry) String() string {
	return fmt.Sprintf("%d (%d) weight: %v, incEdgeKey: %d", e.AdjNode, e.PrepareEdge, e.Weight, e.IncEdgeKey)
}
