package routing

import (
	"math"

	routingutil "gohopper/core/routing/util"
)

// Algorithm name constants matching Java Parameters.Algorithms.
const (
	AlgoDijkstra          = "dijkstra"
	AlgoDijkstraBi        = "dijkstrabi"
	AlgoAStarBi           = "astarbi"
	AlgoDijkstraOneToMany = "dijkstra_one_to_many"
	AlgoAStar             = "astar"
	AlgoAltRoute          = "alternative_route"
)

type AlgorithmOptions struct {
	Algorithm       string
	TraversalMode   routingutil.TraversalMode
	MaxVisitedNodes int
	TimeoutMillis   int64
	Hints           map[string]string
}

func NewAlgorithmOptions() AlgorithmOptions {
	return AlgorithmOptions{
		Algorithm:       AlgoAStarBi,
		TraversalMode:   routingutil.NodeBased,
		MaxVisitedNodes: math.MaxInt,
		TimeoutMillis:   math.MaxInt64,
		Hints:           make(map[string]string),
	}
}
