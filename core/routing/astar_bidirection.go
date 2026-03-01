package routing

import (
	"fmt"

	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// AStarBidirection implements bidirectional A* search using a balanced
// weight approximator for consistent heuristic estimates.
type AStarBidirection struct {
	AbstractBidirAlgo
	weightApprox           *weighting.BalancedWeightApproximator
	stoppingCriterionOffset float64
}

func NewAStarBidirection(graph storage.Graph, w weighting.Weighting, tMode routingutil.TraversalMode) *AStarBidirection {
	a := &AStarBidirection{
		AbstractBidirAlgo: NewAbstractBidirAlgo(graph, w, tMode),
	}
	a.Name = AlgoAStarBi

	defaultApprox := weighting.NewBeelineWeightApproximator(a.NodeAccess, w)
	defaultApprox.SetDistanceCalc(util.DistPlane)
	a.SetApproximation(defaultApprox)

	// Wire hooks
	a.PreInitFn = a.preInit
	a.FinishedFn = a.astarFinished
	a.CreateStartEntryFn = a.astarCreateStartEntry
	a.CreateEntryFn = a.astarCreateEntry
	return a
}

func (a *AStarBidirection) SetApproximation(approx weighting.WeightApproximator) *AStarBidirection {
	a.weightApprox = weighting.NewBalancedWeightApproximator(approx)
	return a
}

func (a *AStarBidirection) GetApproximation() weighting.WeightApproximator {
	return a.weightApprox.GetApproximation()
}

func (a *AStarBidirection) preInit(from int, fromWeight float64, to int, toWeight float64) {
	a.weightApprox.SetFromTo(from, to)
	a.stoppingCriterionOffset = a.weightApprox.Approximate(to, true) + a.weightApprox.GetSlack()
}

func (a *AStarBidirection) astarFinished() bool {
	if a.finishedFrom || a.finishedTo {
		return true
	}
	return a.currFrom.Weight+a.currTo.Weight >= a.BestWeight+a.stoppingCriterionOffset
}

func (a *AStarBidirection) astarCreateStartEntry(node int, weight float64, reverse bool) *SPTEntry {
	heapWeight := weight + a.weightApprox.Approximate(node, reverse)
	return NewSPTEntryWithHeuristic(util.NoEdge, node, heapWeight, weight, nil)
}

func (a *AStarBidirection) astarCreateEntry(edge util.EdgeIteratorState, weight float64, parent *SPTEntry, reverse bool) *SPTEntry {
	neighborNode := edge.GetAdjNode()
	heapWeight := weight + a.weightApprox.Approximate(neighborNode, reverse)
	return NewSPTEntryWithHeuristic(edge.GetEdge(), neighborNode, heapWeight, weight, parent)
}

func (a *AStarBidirection) GetName() string {
	return fmt.Sprintf("%s|%s", AlgoAStarBi, a.weightApprox)
}
