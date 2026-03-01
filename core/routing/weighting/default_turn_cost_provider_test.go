package weighting

import (
	"math"
	"testing"

	"gohopper/core/routing/ev"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// newTurnCostGraph creates a BaseGraph with turn costs enabled and three nodes
// connected by two edges (0->1->2). Returns the graph and its NodeAccess.
func newTurnCostGraph() (*storage.BaseGraph, storage.NodeAccess) {
	g := storage.NewBaseGraphBuilder(4).SetWithTurnCosts(true).CreateGraph()
	na := g.GetNodeAccess()
	na.SetNode(0, 0, 0, 0)
	na.SetNode(1, 1, 1, 0)
	na.SetNode(2, 2, 2, 0)
	g.Edge(0, 1) // edge 0
	g.Edge(1, 2) // edge 1
	return g, na
}

// newTurnRestrictionEnc creates and initialises a BooleanEncodedValue suitable
// for use as a turn restriction flag.
func newTurnRestrictionEnc() *ev.SimpleBooleanEncodedValue {
	cfg := ev.NewInitializerConfig()
	enc := ev.NewSimpleBooleanEncodedValue("turn_restriction")
	enc.Init(cfg)
	return enc
}

func TestDefaultTurnCostProvider_InvalidEdges(t *testing.T) {
	g, na := newTurnCostGraph()
	defer g.Close()

	p := NewDefaultTurnCostProvider(nil, g.TurnCostStorage, na, 40)

	if w := p.CalcTurnWeight(util.NoEdge, 1, 0); w != 0 {
		t.Fatalf("expected 0 for invalid inEdge, got %f", w)
	}
	if w := p.CalcTurnWeight(0, 1, util.NoEdge); w != 0 {
		t.Fatalf("expected 0 for invalid outEdge, got %f", w)
	}
	if w := p.CalcTurnWeight(util.NoEdge, 1, util.NoEdge); w != 0 {
		t.Fatalf("expected 0 for both invalid, got %f", w)
	}
}

func TestDefaultTurnCostProvider_UTurn(t *testing.T) {
	g, na := newTurnCostGraph()
	defer g.Close()

	p := NewDefaultTurnCostProvider(nil, g.TurnCostStorage, na, 40)

	if w := p.CalcTurnWeight(0, 1, 0); w != 40 {
		t.Fatalf("expected 40 for u-turn, got %f", w)
	}
}

func TestDefaultTurnCostProvider_InfiniteUTurn(t *testing.T) {
	g, na := newTurnCostGraph()
	defer g.Close()

	p := NewDefaultTurnCostProvider(nil, g.TurnCostStorage, na, -1)

	w := p.CalcTurnWeight(0, 1, 0)
	if !math.IsInf(w, 1) {
		t.Fatalf("expected +Inf for u-turn with negative uTurnCosts, got %f", w)
	}
}

func TestDefaultTurnCostProvider_Restriction(t *testing.T) {
	g, na := newTurnCostGraph()
	defer g.Close()

	enc := newTurnRestrictionEnc()
	g.TurnCostStorage.SetBool(na, enc, 0, 1, 1, true)

	p := NewDefaultTurnCostProvider(enc, g.TurnCostStorage, na, 40)

	w := p.CalcTurnWeight(0, 1, 1)
	if !math.IsInf(w, 1) {
		t.Fatalf("expected +Inf for restricted turn, got %f", w)
	}
}

func TestDefaultTurnCostProvider_NoRestriction(t *testing.T) {
	g, na := newTurnCostGraph()
	defer g.Close()

	enc := newTurnRestrictionEnc()
	// No restriction set for 0->1->1.

	p := NewDefaultTurnCostProvider(enc, g.TurnCostStorage, na, 40)

	if w := p.CalcTurnWeight(0, 1, 1); w != 0 {
		t.Fatalf("expected 0 for unrestricted turn, got %f", w)
	}
}

func TestDefaultTurnCostProvider_CalcTurnMillis(t *testing.T) {
	g, na := newTurnCostGraph()
	defer g.Close()

	p := NewDefaultTurnCostProvider(nil, g.TurnCostStorage, na, 40)

	if m := p.CalcTurnMillis(0, 1, 1); m != 0 {
		t.Fatalf("expected 0, got %d", m)
	}
	if m := p.CalcTurnMillis(util.NoEdge, 1, 0); m != 0 {
		t.Fatalf("expected 0 for invalid edge, got %d", m)
	}
}
