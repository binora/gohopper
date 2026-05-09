package routing

import (
	routingutil "gohopper/core/routing/util"
	"gohopper/core/storage"
)

// DijkstraBidirectionCHNoSOD is the node-based bidirectional CH Dijkstra
// query algorithm without stall-on-demand.
type DijkstraBidirectionCHNoSOD struct {
	AbstractBidirCHAlgo
}

func NewDijkstraBidirectionCHNoSOD(graph storage.RoutingCHGraph) *DijkstraBidirectionCHNoSOD {
	algo := &DijkstraBidirectionCHNoSOD{
		AbstractBidirCHAlgo: NewAbstractBidirCHAlgo(graph, routingutil.NodeBased),
	}
	algo.Name = "dijkstrabi|ch|no_sod"
	algo.CreateStartEntryFn = newNodeBasedCHStartEntry
	algo.CreateCHEntryFn = newNodeBasedCHEntry
	return algo
}

func newNodeBasedCHStartEntry(node int, weight float64, _ bool) *SPTEntry {
	return NewSPTEntry(node, weight)
}

func newNodeBasedCHEntry(edge, adjNode, _ int, weight float64, parent *SPTEntry, _ bool) *SPTEntry {
	return NewSPTEntryFull(edge, adjNode, weight, parent)
}
