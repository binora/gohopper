package routing

import (
	"fmt"
	"strings"

	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
)

type RoutingAlgorithmFactorySimple struct{}

func (f *RoutingAlgorithmFactorySimple) CreateAlgo(g storage.Graph, w weighting.Weighting, opts AlgorithmOptions) RoutingAlgorithm {
	algoStr := strings.ToLower(opts.Algorithm)

	var ra RoutingAlgorithm
	switch algoStr {
	case AlgoDijkstra:
		ra = NewDijkstra(g, w, opts.TraversalMode)
	case AlgoDijkstraBi:
		ra = NewDijkstraBidirectionRef(g, w, opts.TraversalMode)
	case AlgoAStar:
		ra = NewAStar(g, w, opts.TraversalMode)
	case AlgoAStarBi:
		ra = NewAStarBidirection(g, w, opts.TraversalMode)
	case "", AlgoDijkstraOneToMany, AlgoAltRoute:
		panic(fmt.Sprintf("algorithm %q not yet implemented", algoStr))
	default:
		panic(fmt.Sprintf("algorithm %q not found", algoStr))
	}

	ra.SetMaxVisitedNodes(opts.MaxVisitedNodes)
	ra.SetTimeoutMillis(opts.TimeoutMillis)
	return ra
}
