package routing

import (
	routingutil "gohopper/core/routing/util"
	"gohopper/core/storage"
)

// DijkstraBidirectionEdgeCHNoSOD runs bidirectional Dijkstra on edge-based CH
// graphs without stall-on-demand.
type DijkstraBidirectionEdgeCHNoSOD struct {
	AbstractBidirCHAlgo
}

func NewDijkstraBidirectionEdgeCHNoSOD(graph storage.RoutingCHGraph) *DijkstraBidirectionEdgeCHNoSOD {
	if !graph.IsEdgeBased() {
		panic("edge-based CH algorithms only work with edge-based CH graphs")
	}
	d := &DijkstraBidirectionEdgeCHNoSOD{
		AbstractBidirCHAlgo: NewAbstractBidirCHAlgo(graph, routingutil.EdgeBased),
	}
	d.Name = "dijkstrabi|ch|edge_based|no_sod"
	d.CreateStartEntryFn = d.createStartEntry
	d.CreateCHEntryFn = d.createEntry
	return d
}

func (d *DijkstraBidirectionEdgeCHNoSOD) createStartEntry(node int, weight float64, _ bool) *SPTEntry {
	return NewSPTEntry(node, weight)
}

func (d *DijkstraBidirectionEdgeCHNoSOD) createEntry(edge, adjNode, incEdge int, weight float64, parent *SPTEntry, _ bool) *SPTEntry {
	return newSPTEntryWithIncEdge(edge, incEdge, adjNode, weight, parent)
}
