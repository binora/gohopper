package routing

import (
	"math"
	"testing"

	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
)

func newBidirPathExtractorGraph(speedEnc ev.DecimalEncodedValue, turnCostEnc ev.DecimalEncodedValue) (*storage.BaseGraph, int) {
	em := routingutil.Start().
		Add(speedEnc).
		AddTurnCostEncodedValue(turnCostEnc).
		Build()
	g := storage.NewBaseGraphBuilder(em.BytesForFlags).
		SetWithTurnCosts(true).
		CreateGraph()
	return g, em.BytesForFlags
}

func TestDefaultBidirPathExtractor_Extract(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	turnCostEnc := ev.TurnCostCreate("car", 10)
	g, _ := newBidirPathExtractorGraph(speedEnc, turnCostEnc)
	t.Cleanup(func() { g.Close() })

	g.Edge(1, 2).SetDistance(10).SetDecimalBothDir(speedEnc, 60, 60)

	fwdEntry := NewSPTEntryFull(0, 2, 0, NewSPTEntry(1, 10))
	bwdEntry := NewSPTEntry(2, 0)

	p := ExtractBidirPath(g, weighting.NewSpeedWeighting(speedEnc), fwdEntry, bwdEntry, 0)
	assertNodesEqual(t, []int{1, 2}, p)
	assertDistEquals(t, 10, p.Distance, 1e-4, p.String())
}

func TestDefaultBidirPathExtractor_Extract2(t *testing.T) {
	// 1->2->3
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	turnCostEnc := ev.TurnCostCreate("car", 10)
	g, _ := newBidirPathExtractorGraph(speedEnc, turnCostEnc)
	t.Cleanup(func() { g.Close() })

	g.Edge(1, 2).SetDistance(10).SetDecimalBothDir(speedEnc, 10, 0)
	g.Edge(2, 3).SetDistance(20).SetDecimalBothDir(speedEnc, 10, 0)

	// add turn cost of 5 at node 2 where fwd&bwd searches meet (edge 0 -> node 2 -> edge 1)
	tcs := g.GetTurnCostStorage()
	na := g.GetNodeAccess()
	tcs.SetDecimal(na, turnCostEnc, 0, 2, 1, 5)

	fwdEntry := NewSPTEntryFull(0, 2, 0.6, NewSPTEntry(1, 0))
	bwdEntry := NewSPTEntryFull(1, 2, 1.2, NewSPTEntry(3, 0))

	w := weighting.NewSpeedWeightingWithTurnCosts(speedEnc, turnCostEnc, tcs, na, math.Inf(1))
	p := ExtractBidirPath(g, w, fwdEntry, bwdEntry, 0)
	p.SetWeight(5 + 3)

	assertNodesEqual(t, []int{1, 2, 3}, p)
	assertDistEquals(t, 30, p.Distance, 1e-4, p.String())
	assertDistEquals(t, 8, p.Weight, 1e-4, p.String())
	if p.Time != 8000 {
		t.Fatalf("expected time 8000, got %d", p.Time)
	}
}
