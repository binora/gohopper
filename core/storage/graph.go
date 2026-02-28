package storage

import (
	"gohopper/core/util"

	routingutil "gohopper/core/routing/util"
)

// AllEdgesIterator iterates over all edges of the graph.
type AllEdgesIterator interface {
	util.EdgeIterator
	Length() int
}

// Graph provides access to the graph data structure.
type Graph interface {
	GetBaseGraph() *BaseGraph
	GetNodes() int
	GetEdges() int
	GetNodeAccess() NodeAccess
	GetBounds() util.BBox
	Edge(a, b int) util.EdgeIteratorState
	GetEdgeIteratorState(edgeID, adjNode int) util.EdgeIteratorState
	GetEdgeIteratorStateForKey(edgeKey int) util.EdgeIteratorState
	GetOtherNode(edge, node int) int
	IsAdjacentToNode(edge, node int) bool
	GetAllEdges() AllEdgesIterator
	CreateEdgeExplorer(filter routingutil.EdgeFilter) util.EdgeExplorer
	GetTurnCostStorage() *TurnCostStorage
}
