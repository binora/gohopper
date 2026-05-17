package routing_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gohopper/core/routing"
	"gohopper/core/routing/ch"
	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	webapi "gohopper/web-api"
)

// altRouteCHTestGraph builds the Java AlternativeRouteCHTest test graph (L41-75):
//
//	      9      11
//	     /\     /  \
//	    1  2-3-4-10-12
//	    \   /   \
//	    5--6-7---8
//
// All edges have the same length so the locality test passes for the three
// natural alternatives.
func altRouteCHTestGraph(t *testing.T) (*storage.BaseGraph, ev.DecimalEncodedValue) {
	t.Helper()
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, false)
	em := routingutil.Start().Add(speedEnc).Build()
	g := storage.NewBaseGraphBuilder(em.BytesForFlags).CreateGraph()
	edges := [][2]int{
		{5, 6}, {6, 3}, {3, 4}, {4, 10},
		{6, 7}, {7, 8}, {8, 4},
		{5, 1}, {1, 9}, {9, 2}, {2, 3},
		{4, 11}, {11, 12}, {12, 10},
	}
	dists := []float64{10000, 10000, 10000, 10000, 10000, 10000, 10000, 10000, 10000, 10000, 10000, 9000, 9000, 10000}
	for i, e := range edges {
		g.Edge(e[0], e[1]).SetDistance(dists[i]).SetDecimal(speedEnc, 60)
	}
	g.Freeze()
	return g, speedEnc
}

// altRouteCHPrepare freezes the graph and builds a node-based RoutingCHGraph
// with the Java-pinned contraction order so fwd/bwd trees meet on all four
// 5→10 paths.
func altRouteCHPrepare(t *testing.T, g *storage.BaseGraph, speedEnc ev.DecimalEncodedValue) storage.RoutingCHGraph {
	t.Helper()
	w := weighting.NewSpeedWeighting(speedEnc)
	chConfig := ch.NewCHConfigNodeBased("p", w)
	order := []int{0, 10, 12, 4, 3, 2, 5, 1, 6, 7, 8, 9, 11}
	prep := ch.FromGraph(g, chConfig).UseFixedNodeOrdering(ch.NodeOrderingFromArray(order...))
	res := prep.DoWork()
	return storage.NewRoutingCHGraph(g, res.GetCHStorage(), chConfig.GetWeighting())
}

// TestAlternativeRouteCH_CalcAlternatives ports Java AlternativeRouteCHTest.testCalcAlternatives.
func TestAlternativeRouteCH_CalcAlternatives(t *testing.T) {
	g, speedEnc := altRouteCHTestGraph(t)
	hints := webapi.NewPMap().
		PutObject("alternative_route.max_weight_factor", 2.3).
		PutObject("alternative_route.local_optimality_factor", 0.5).
		PutObject("alternative_route.max_paths", 4)
	chGraph := altRouteCHPrepare(t, g, speedEnc)
	algo := routing.NewAlternativeRouteCH(chGraph, hints)
	algo.SetPathExtractorSupplier(func() routing.BidirPathExtractor {
		return ch.NewNodeBasedCHBidirPathExtractor(chGraph)
	})
	alts := algo.CalcAlternatives(5, 10)
	assert.Equal(t, 3, len(alts))
}

// TestAlternativeRouteCH_RelaxMaximumStretch ports Java AlternativeRouteCHTest.testRelaxMaximumStretch.
func TestAlternativeRouteCH_RelaxMaximumStretch(t *testing.T) {
	g, speedEnc := altRouteCHTestGraph(t)
	hints := webapi.NewPMap().
		PutObject("alternative_route.max_weight_factor", 4).
		PutObject("alternative_route.local_optimality_factor", 0.5).
		PutObject("alternative_route.max_paths", 4)
	chGraph := altRouteCHPrepare(t, g, speedEnc)
	algo := routing.NewAlternativeRouteCH(chGraph, hints)
	algo.SetPathExtractorSupplier(func() routing.BidirPathExtractor {
		return ch.NewNodeBasedCHBidirPathExtractor(chGraph)
	})
	alts := algo.CalcAlternatives(5, 10)
	assert.Equal(t, 4, len(alts))
}
