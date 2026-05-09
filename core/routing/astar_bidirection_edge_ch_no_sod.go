package routing

import (
	"fmt"

	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// AStarBidirectionEdgeCHNoSOD runs the edge-based CH query variant GraphHopper
// uses by default. Like the Java implementation, it uses the heuristic for
// pruning, but not for heap ordering.
type AStarBidirectionEdgeCHNoSOD struct {
	AbstractBidirCHAlgo
	weightApprox *weighting.BalancedWeightApproximator
}

func NewAStarBidirectionEdgeCHNoSOD(graph storage.RoutingCHGraph) *AStarBidirectionEdgeCHNoSOD {
	if !graph.IsEdgeBased() {
		panic("edge-based CH algorithms only work with edge-based CH graphs")
	}
	a := &AStarBidirectionEdgeCHNoSOD{
		AbstractBidirCHAlgo: NewAbstractBidirCHAlgo(graph, routingutil.EdgeBased),
	}
	a.Name = "astarbi|ch|edge_based|no_sod"

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
	a.FromEntryCanBeSkipped = a.fromEntryCanBeSkipped
	a.ToEntryCanBeSkipped = a.toEntryCanBeSkipped
	return a
}

func (a *AStarBidirectionEdgeCHNoSOD) SetApproximation(approx weighting.WeightApproximator) *AStarBidirectionEdgeCHNoSOD {
	a.weightApprox = weighting.NewBalancedWeightApproximator(approx)
	return a
}

func (a *AStarBidirectionEdgeCHNoSOD) GetApproximation() weighting.WeightApproximator {
	return a.weightApprox.GetApproximation()
}

func (a *AStarBidirectionEdgeCHNoSOD) preInit(from int, _ float64, to int, _ float64) {
	a.weightApprox.SetFromTo(from, to)
}

func (a *AStarBidirectionEdgeCHNoSOD) fromEntryCanBeSkipped() bool {
	return a.currFrom.Weight+a.weightApprox.Approximate(a.currFrom.AdjNode, false) > a.BestWeight
}

func (a *AStarBidirectionEdgeCHNoSOD) toEntryCanBeSkipped() bool {
	return a.currTo.Weight+a.weightApprox.Approximate(a.currTo.AdjNode, true) > a.BestWeight
}

func (a *AStarBidirectionEdgeCHNoSOD) createStartEntry(node int, weight float64, _ bool) *SPTEntry {
	return NewSPTEntryWithHeuristic(util.NoEdge, node, weight, weight, nil)
}

func (a *AStarBidirectionEdgeCHNoSOD) createEntry(edge, adjNode, incEdge int, weight float64, parent *SPTEntry, _ bool) *SPTEntry {
	return newSPTEntryWithIncEdge(edge, incEdge, adjNode, weight, parent)
}
