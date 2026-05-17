package routing

import (
	"fmt"

	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// AStarBidirectionCH is the node-based bidirectional CH A* query algorithm.
// Unlike the edge-based variant, the heuristic is folded into the heap weight
// so node ordering follows A* directly.
type AStarBidirectionCH struct {
	AbstractBidirCHAlgo
	weightApprox *weighting.BalancedWeightApproximator
}

func NewAStarBidirectionCH(graph storage.RoutingCHGraph) *AStarBidirectionCH {
	a := &AStarBidirectionCH{
		AbstractBidirCHAlgo: NewAbstractBidirCHAlgo(graph, routingutil.NodeBased),
	}
	a.Name = "astarbi|ch"

	w, ok := graph.GetWeighting().(weighting.Weighting)
	if !ok {
		panic(fmt.Sprintf("CH weighting %T does not implement weighting.Weighting", graph.GetWeighting()))
	}
	defaultApprox := weighting.NewBeelineWeightApproximator(graph.GetBaseGraph().GetNodeAccess(), w)
	defaultApprox.SetDistanceCalc(util.DistPlane)
	a.SetApproximation(defaultApprox)

	a.PreInitFn = a.preInit
	a.CreateStartEntryFn = a.createStartEntry
	a.CreateCHEntryFn = a.createEntry
	return a
}

func (a *AStarBidirectionCH) SetApproximation(approx weighting.WeightApproximator) *AStarBidirectionCH {
	a.weightApprox = weighting.NewBalancedWeightApproximator(approx)
	return a
}

func (a *AStarBidirectionCH) GetApproximation() weighting.WeightApproximator {
	return a.weightApprox.GetApproximation()
}

func (a *AStarBidirectionCH) preInit(from int, _ float64, to int, _ float64) {
	a.weightApprox.SetFromTo(from, to)
}

func (a *AStarBidirectionCH) createStartEntry(node int, weight float64, reverse bool) *SPTEntry {
	heapWeight := weight + a.weightApprox.Approximate(node, reverse)
	return NewSPTEntryWithHeuristic(util.NoEdge, node, heapWeight, weight, nil)
}

func (a *AStarBidirectionCH) createEntry(edge, adjNode, _ int, weight float64, parent *SPTEntry, reverse bool) *SPTEntry {
	heapWeight := weight + a.weightApprox.Approximate(adjNode, reverse)
	return NewSPTEntryWithHeuristic(edge, adjNode, heapWeight, weight, parent)
}
