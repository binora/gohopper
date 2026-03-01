package routing

import (
	"strings"

	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
)

// RoutingAlgorithmFactorySimple creates standard routing algorithms without preparation.
// It mirrors Java's RoutingAlgorithmFactorySimple.
type RoutingAlgorithmFactorySimple struct{}

// CreateAlgo creates a routing algorithm based on opts.Algorithm.
func (f *RoutingAlgorithmFactorySimple) CreateAlgo(g storage.Graph, w weighting.Weighting, opts AlgorithmOptions) RoutingAlgorithm {
	algoStr := strings.ToLower(opts.Algorithm)

	var ra RoutingAlgorithm
	switch algoStr {
	case AlgoDijkstra:
		ra = NewDijkstra(g, w, opts.TraversalMode)
	case AlgoDijkstraBi:
		panic("Algorithm " + algoStr + " not yet implemented")
	case AlgoAStar:
		panic("Algorithm " + algoStr + " not yet implemented")
	case AlgoAStarBi, "":
		panic("Algorithm " + AlgoAStarBi + " not yet implemented")
	case AlgoDijkstraOneToMany:
		panic("Algorithm " + algoStr + " not yet implemented")
	case AlgoAltRoute:
		panic("Algorithm " + algoStr + " not yet implemented")
	default:
		panic("Algorithm " + algoStr + " not found in RoutingAlgorithmFactorySimple")
	}

	ra.SetMaxVisitedNodes(opts.MaxVisitedNodes)
	ra.SetTimeoutMillis(opts.TimeoutMillis)
	return ra
}
