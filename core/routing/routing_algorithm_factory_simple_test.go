package routing

import (
	"math"
	"testing"

	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
)

// factoryFixture holds shared state for RoutingAlgorithmFactorySimple tests.
type factoryFixture struct {
	speedEnc      ev.DecimalEncodedValue
	weighting     weighting.Weighting
	bytesForFlags int
}

func newFactoryFixture() *factoryFixture {
	speedEnc := ev.NewDecimalEncodedValueImpl("car_speed", 5, 5, true)
	cfg := ev.NewInitializerConfig()
	speedEnc.Init(cfg)
	return &factoryFixture{
		speedEnc:      speedEnc,
		weighting:     weighting.NewSpeedWeighting(speedEnc),
		bytesForFlags: cfg.GetRequiredBytes(),
	}
}

func (f *factoryFixture) createGraph(t *testing.T) *storage.BaseGraph {
	t.Helper()
	g := storage.NewBaseGraphBuilder(f.bytesForFlags).CreateGraph()
	t.Cleanup(func() { g.Close() })
	return g
}

// TestRoutingAlgorithmFactorySimple_Dijkstra verifies that the factory creates
// a Dijkstra algorithm and that GetName returns "dijkstra".
func TestRoutingAlgorithmFactorySimple_Dijkstra(t *testing.T) {
	ff := newFactoryFixture()
	g := ff.createGraph(t)
	g.Edge(0, 1).SetDistance(10).SetDecimalBothDir(ff.speedEnc, 60, 60)

	factory := &RoutingAlgorithmFactorySimple{}
	opts := AlgorithmOptions{
		Algorithm:       AlgoDijkstra,
		TraversalMode:   routingutil.NodeBased,
		MaxVisitedNodes: math.MaxInt,
		TimeoutMillis:   math.MaxInt64,
	}
	algo := factory.CreateAlgo(g, ff.weighting, opts)

	if algo.GetName() != AlgoDijkstra {
		t.Fatalf("expected algorithm name %q, got %q", AlgoDijkstra, algo.GetName())
	}
}

// TestRoutingAlgorithmFactorySimple_UnknownAlgo verifies that the factory panics
// for an unknown algorithm name.
func TestRoutingAlgorithmFactorySimple_UnknownAlgo(t *testing.T) {
	ff := newFactoryFixture()
	g := ff.createGraph(t)

	factory := &RoutingAlgorithmFactorySimple{}
	opts := AlgorithmOptions{
		Algorithm:       "unknown_algo",
		TraversalMode:   routingutil.NodeBased,
		MaxVisitedNodes: math.MaxInt,
		TimeoutMillis:   math.MaxInt64,
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for unknown algorithm")
		}
	}()
	factory.CreateAlgo(g, ff.weighting, opts)
}

// TestRoutingAlgorithmFactorySimple_MaxVisitedNodes verifies that the factory
// applies MaxVisitedNodes from AlgorithmOptions to the created algorithm.
func TestRoutingAlgorithmFactorySimple_MaxVisitedNodes(t *testing.T) {
	ff := newFactoryFixture()
	g := ff.createGraph(t)
	initBiGraph(g, ff.speedEnc)

	factory := &RoutingAlgorithmFactorySimple{}

	// With unlimited visited nodes, path should be found.
	optsUnlimited := AlgorithmOptions{
		Algorithm:       AlgoDijkstra,
		TraversalMode:   routingutil.NodeBased,
		MaxVisitedNodes: math.MaxInt,
		TimeoutMillis:   math.MaxInt64,
	}
	algo := factory.CreateAlgo(g, ff.weighting, optsUnlimited)
	p := algo.CalcPath(0, 4)
	if !p.Found {
		t.Fatal("expected path found with unlimited visited nodes")
	}

	// With very limited visited nodes, path should not be found.
	optsLimited := AlgorithmOptions{
		Algorithm:       AlgoDijkstra,
		TraversalMode:   routingutil.NodeBased,
		MaxVisitedNodes: 3,
		TimeoutMillis:   math.MaxInt64,
	}
	algo2 := factory.CreateAlgo(g, ff.weighting, optsLimited)
	p2 := algo2.CalcPath(0, 4)
	if p2.Found {
		t.Fatal("expected no path found with maxVisitedNodes=3")
	}
}
