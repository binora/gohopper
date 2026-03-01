package routing

import (
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
)

// RoutingAlgorithmFactory creates RoutingAlgorithm instances.
type RoutingAlgorithmFactory interface {
	// CreateAlgo creates a routing algorithm for the given graph, weighting, and options.
	CreateAlgo(g storage.Graph, w weighting.Weighting, opts AlgorithmOptions) RoutingAlgorithm
}
