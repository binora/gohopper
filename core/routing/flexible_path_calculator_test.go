package routing

import (
	"math"
	"strings"
	"testing"

	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
)

// fpcFixture holds shared state for FlexiblePathCalculator tests.
type fpcFixture struct {
	speedEnc      ev.DecimalEncodedValue
	weighting     weighting.Weighting
	bytesForFlags int
}

func newFPCFixture() *fpcFixture {
	speedEnc := ev.NewDecimalEncodedValueImpl("car_speed", 5, 5, true)
	cfg := ev.NewInitializerConfig()
	speedEnc.Init(cfg)
	return &fpcFixture{
		speedEnc:      speedEnc,
		weighting:     weighting.NewSpeedWeighting(speedEnc),
		bytesForFlags: cfg.GetRequiredBytes(),
	}
}

func (f *fpcFixture) createGraph(t *testing.T) *storage.BaseGraph {
	t.Helper()
	g := storage.NewBaseGraphBuilder(f.bytesForFlags).CreateGraph()
	t.Cleanup(func() { g.Close() })
	return g
}

func (f *fpcFixture) createCalculator(g storage.Graph) *FlexiblePathCalculator {
	opts := AlgorithmOptions{
		Algorithm:       AlgoDijkstra,
		TraversalMode:   routingutil.NodeBased,
		MaxVisitedNodes: math.MaxInt,
		TimeoutMillis:   math.MaxInt64,
	}
	factory := &RoutingAlgorithmFactorySimple{}
	return NewFlexiblePathCalculator(g, factory, f.weighting, opts)
}

// TestFlexiblePathCalculator_CalcPaths builds a small graph and verifies
// that FlexiblePathCalculator finds a valid path.
func TestFlexiblePathCalculator_CalcPaths(t *testing.T) {
	f := newFPCFixture()
	g := f.createGraph(t)

	// Build a simple graph: 0--1--2--3
	g.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 60, 60)
	g.Edge(1, 2).SetDistance(20).SetDecimalBothDir(f.speedEnc, 60, 60)
	g.Edge(2, 3).SetDistance(30).SetDecimalBothDir(f.speedEnc, 60, 60)

	calc := f.createCalculator(g)
	paths := calc.CalcPaths(0, 3, NewEdgeRestrictions())

	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}
	p := paths[0]
	if !p.Found {
		t.Fatal("expected path to be found")
	}

	nodes := p.CalcNodes()
	expected := []int{0, 1, 2, 3}
	if len(nodes) != len(expected) {
		t.Fatalf("expected nodes %v, got %v", expected, nodes)
	}
	for i, n := range expected {
		if nodes[i] != n {
			t.Fatalf("node mismatch at index %d: expected %d, got %d. full: %v", i, n, nodes[i], nodes)
		}
	}
}

// TestFlexiblePathCalculator_GetVisitedNodes verifies that the visited node
// count is propagated from the underlying algorithm.
func TestFlexiblePathCalculator_GetVisitedNodes(t *testing.T) {
	f := newFPCFixture()
	g := f.createGraph(t)

	// 0--1--2
	g.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 60, 60)
	g.Edge(1, 2).SetDistance(20).SetDecimalBothDir(f.speedEnc, 60, 60)

	calc := f.createCalculator(g)
	calc.CalcPaths(0, 2, NewEdgeRestrictions())

	visited := calc.GetVisitedNodes()
	if visited <= 0 {
		t.Fatalf("expected visited nodes > 0, got %d", visited)
	}
}

// TestFlexiblePathCalculator_DebugString verifies that the debug string is
// populated after a path calculation.
func TestFlexiblePathCalculator_DebugString(t *testing.T) {
	f := newFPCFixture()
	g := f.createGraph(t)

	// 0--1
	g.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 60, 60)

	calc := f.createCalculator(g)
	calc.CalcPaths(0, 1, NewEdgeRestrictions())

	debug := calc.GetDebugString()
	if debug == "" {
		t.Fatal("expected non-empty debug string")
	}
	if !strings.Contains(debug, "algoInit:") {
		t.Fatalf("expected debug string to contain 'algoInit:', got %q", debug)
	}
	if !strings.Contains(debug, AlgoDijkstra+"-routing:") {
		t.Fatalf("expected debug string to contain '%s-routing:', got %q", AlgoDijkstra, debug)
	}
}
