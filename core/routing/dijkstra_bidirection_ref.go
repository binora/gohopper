package routing

import (
	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
)

// DijkstraBidirectionRef calculates shortest paths using a reference
// bidirectional Dijkstra implementation.
type DijkstraBidirectionRef struct {
	AbstractBidirAlgo
}

func NewDijkstraBidirectionRef(graph storage.Graph, w weighting.Weighting, tMode routingutil.TraversalMode) *DijkstraBidirectionRef {
	d := &DijkstraBidirectionRef{
		AbstractBidirAlgo: NewAbstractBidirAlgo(graph, w, tMode),
	}
	d.Name = AlgoDijkstraBi
	return d
}
