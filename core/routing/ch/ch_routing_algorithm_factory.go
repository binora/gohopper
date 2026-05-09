package ch

import (
	"fmt"
	"strings"

	"gohopper/core/routing"
	"gohopper/core/storage"
	webapi "gohopper/web-api"
)

const (
	chRoutingAlgorithmKey = "algorithm"
	chMaxVisitedNodesKey  = "max_visited_nodes"
	chTimeoutMillisKey    = "timeout_ms"
	chStallOnDemandKey    = "stall_on_demand"
)

// CHRoutingAlgorithmFactory creates routing algorithms for prepared CH graphs.
type CHRoutingAlgorithmFactory struct {
	routingCHGraph storage.RoutingCHGraph
}

type nodeBasedCHAlgo interface {
	routing.EdgeToEdgeRoutingAlgorithm
	SetPathExtractorSupplier(func() routing.BidirPathExtractor)
}

func NewCHRoutingAlgorithmFactory(routingCHGraph storage.RoutingCHGraph) *CHRoutingAlgorithmFactory {
	return &CHRoutingAlgorithmFactory{routingCHGraph: routingCHGraph}
}

func (f *CHRoutingAlgorithmFactory) CreateAlgo(opts webapi.PMap) routing.EdgeToEdgeRoutingAlgorithm {
	var algo routing.EdgeToEdgeRoutingAlgorithm
	if f.routingCHGraph.IsEdgeBased() {
		algo = f.createAlgoEdgeBased(opts)
	} else {
		algo = f.createAlgoNodeBased(opts)
	}
	if opts.Has(chMaxVisitedNodesKey) {
		algo.SetMaxVisitedNodes(opts.GetInt(chMaxVisitedNodesKey, 0))
	}
	if opts.Has(chTimeoutMillisKey) {
		algo.SetTimeoutMillis(int64(opts.GetInt(chTimeoutMillisKey, 0)))
	}
	return algo
}

func (f *CHRoutingAlgorithmFactory) createAlgoNodeBased(opts webapi.PMap) routing.EdgeToEdgeRoutingAlgorithm {
	algoName := strings.ToLower(opts.GetString(chRoutingAlgorithmKey, routing.AlgoDijkstraBi))
	if algoName == "" {
		algoName = routing.AlgoDijkstraBi
	}

	switch algoName {
	case routing.AlgoDijkstraBi:
		if opts.GetBool(chStallOnDemandKey, true) {
			return f.withNodeBasedPathExtractor(routing.NewDijkstraBidirectionCH(f.routingCHGraph))
		}
		return f.withNodeBasedPathExtractor(routing.NewDijkstraBidirectionCHNoSOD(f.routingCHGraph))
	case routing.AlgoAStarBi, routing.AlgoAltRoute:
		panic(fmt.Sprintf("algorithm %q not yet supported for node-based Contraction Hierarchies", algoName))
	default:
		panic(fmt.Sprintf("algorithm %q not supported for node-based Contraction Hierarchies", algoName))
	}
}

func (f *CHRoutingAlgorithmFactory) withNodeBasedPathExtractor(algo nodeBasedCHAlgo) routing.EdgeToEdgeRoutingAlgorithm {
	algo.SetPathExtractorSupplier(func() routing.BidirPathExtractor {
		return NewNodeBasedCHBidirPathExtractor(f.routingCHGraph)
	})
	return algo
}

func (f *CHRoutingAlgorithmFactory) createAlgoEdgeBased(opts webapi.PMap) routing.EdgeToEdgeRoutingAlgorithm {
	algoName := strings.ToLower(opts.GetString(chRoutingAlgorithmKey, routing.AlgoAStarBi))
	if algoName == "" {
		algoName = routing.AlgoAStarBi
	}
	panic(fmt.Sprintf("algorithm %q not yet supported for edge-based Contraction Hierarchies", algoName))
}
