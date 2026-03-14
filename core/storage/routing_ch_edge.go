package storage

import "gohopper/core/util"

// CHWeighting mirrors weighting.Weighting to avoid an import cycle.
type CHWeighting interface {
	CalcMinWeightPerDistance() float64
	CalcEdgeWeight(edgeState util.EdgeIteratorState, reverse bool) float64
	CalcEdgeMillis(edgeState util.EdgeIteratorState, reverse bool) int64
	CalcTurnWeight(inEdge, viaNode, outEdge int) float64
	CalcTurnMillis(inEdge, viaNode, outEdge int) int64
	HasTurnCosts() bool
	GetName() string
}

type RoutingCHEdgeIteratorState interface {
	GetEdge() int
	GetOrigEdge() int
	GetOrigEdgeKeyFirst() int
	GetOrigEdgeKeyLast() int
	GetBaseNode() int
	GetAdjNode() int
	IsShortcut() bool
	GetSkippedEdge1() int
	GetSkippedEdge2() int
	GetWeight(reverse bool) float64
}

type RoutingCHEdgeIterator interface {
	RoutingCHEdgeIteratorState
	Next() bool
}

type RoutingCHEdgeExplorer interface {
	SetBaseNode(baseNode int) RoutingCHEdgeIterator
}

type CHEdgeFilter func(RoutingCHEdgeIteratorState) bool

var AllCHEdges CHEdgeFilter = func(_ RoutingCHEdgeIteratorState) bool { return true }
