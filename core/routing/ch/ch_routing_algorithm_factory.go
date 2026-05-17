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

type chAlgoWithPathExtractor interface {
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
	algoName := chAlgoName(opts, routing.AlgoDijkstraBi)

	switch algoName {
	case routing.AlgoDijkstraBi:
		if opts.GetBool(chStallOnDemandKey, true) {
			return f.withNodeBasedPathExtractor(routing.NewDijkstraBidirectionCH(f.routingCHGraph))
		}
		return f.withNodeBasedPathExtractor(routing.NewDijkstraBidirectionCHNoSOD(f.routingCHGraph))
	case routing.AlgoAStarBi:
		return f.withNodeBasedPathExtractor(routing.NewAStarBidirectionCH(f.routingCHGraph))
	case routing.AlgoAltRoute:
		panic(fmt.Sprintf("algorithm %q not yet supported for node-based Contraction Hierarchies (gohopper-alx)", algoName))
	default:
		panic(fmt.Sprintf("algorithm %q not supported for node-based Contraction Hierarchies", algoName))
	}
}

func (f *CHRoutingAlgorithmFactory) withNodeBasedPathExtractor(algo chAlgoWithPathExtractor) routing.EdgeToEdgeRoutingAlgorithm {
	algo.SetPathExtractorSupplier(func() routing.BidirPathExtractor {
		return NewNodeBasedCHBidirPathExtractor(f.routingCHGraph)
	})
	return algo
}

func (f *CHRoutingAlgorithmFactory) createAlgoEdgeBased(opts webapi.PMap) routing.EdgeToEdgeRoutingAlgorithm {
	algoName := chAlgoName(opts, routing.AlgoAStarBi)

	switch algoName {
	case routing.AlgoAStarBi:
		return f.withEdgeBasedPathExtractor(routing.NewAStarBidirectionEdgeCHNoSOD(f.routingCHGraph))
	case routing.AlgoDijkstraBi:
		return f.withEdgeBasedPathExtractor(routing.NewDijkstraBidirectionEdgeCHNoSOD(f.routingCHGraph))
	case routing.AlgoAltRoute:
		panic(fmt.Sprintf("algorithm %q not yet supported for edge-based Contraction Hierarchies (gohopper-alx)", algoName))
	default:
		panic(fmt.Sprintf("algorithm %q not supported for edge-based Contraction Hierarchies", algoName))
	}
}

func (f *CHRoutingAlgorithmFactory) withEdgeBasedPathExtractor(algo chAlgoWithPathExtractor) routing.EdgeToEdgeRoutingAlgorithm {
	algo.SetPathExtractorSupplier(func() routing.BidirPathExtractor {
		return NewEdgeBasedCHBidirPathExtractor(f.routingCHGraph)
	})
	return algo
}

func chAlgoName(opts webapi.PMap, defaultAlgo string) string {
	algoName := strings.ToLower(opts.GetString(chRoutingAlgorithmKey, defaultAlgo))
	if algoName == "" {
		return defaultAlgo
	}
	return algoName
}
