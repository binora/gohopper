package routing

import (
	"math"
	"testing"

	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
)

type bidirFixture struct {
	name              string
	traversalMode     routingutil.TraversalMode
	speedEnc          ev.DecimalEncodedValue
	defaultWeighting  weighting.Weighting
	defaultMaxVisited int
	bytesForFlags     int
}

func newBidirFixture(name string, tMode routingutil.TraversalMode) *bidirFixture {
	speedEnc := ev.NewDecimalEncodedValueImpl("car_speed", 5, 5, true)
	cfg := ev.NewInitializerConfig()
	speedEnc.Init(cfg)
	return &bidirFixture{
		name:              name,
		traversalMode:     tMode,
		speedEnc:          speedEnc,
		defaultWeighting:  weighting.NewSpeedWeighting(speedEnc),
		defaultMaxVisited: math.MaxInt,
		bytesForFlags:     cfg.GetRequiredBytes(),
	}
}

func (f *bidirFixture) createGHStorage(t *testing.T) *storage.BaseGraph {
	t.Helper()
	b := storage.NewBaseGraphBuilder(f.bytesForFlags)
	if f.traversalMode.IsEdgeBased() {
		b.SetWithTurnCosts(true)
	}
	g := b.CreateGraph()
	t.Cleanup(func() { g.Close() })
	return g
}

func (f *bidirFixture) calcPath(g *storage.BaseGraph, from, to int) *Path {
	return f.calcPathWithWeighting(g, f.defaultWeighting, f.defaultMaxVisited, from, to)
}

func (f *bidirFixture) calcPathWithWeighting(g *storage.BaseGraph, w weighting.Weighting, maxVisitedNodes int, from, to int) *Path {
	algo := NewDijkstraBidirectionRef(g, w, f.traversalMode)
	algo.SetMaxVisitedNodes(maxVisitedNodes)
	return algo.CalcPath(from, to)
}

func TestDijkstraBidirectionRef(t *testing.T) {
	fixtures := []*bidirFixture{
		newBidirFixture("node_based", routingutil.NodeBased),
		newBidirFixture("edge_based", routingutil.EdgeBased),
	}

	for _, f := range fixtures {
		t.Run(f.name, func(t *testing.T) {
			t.Run("CalcShortestPath", func(t *testing.T) {
				g := f.createGHStorage(t)
				initTestStorage(g, f.speedEnc)
				p := f.calcPath(g, 0, 7)
				assertNodesEqual(t, []int{0, 4, 5, 7}, p)
				assertDistEquals(t, 62.1, p.Distance, 0.1, p.String())
			})

			t.Run("SourceEqualsTarget", func(t *testing.T) {
				g := f.createGHStorage(t)
				g.Edge(0, 1).SetDistance(1).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(1, 2).SetDistance(2).SetDecimalBothDir(f.speedEnc, 60, 60)
				p := f.calcPath(g, 0, 0)
				assertPathFromEqualsTo(t, p, 0)
			})

			t.Run("SimpleAlternative", func(t *testing.T) {
				g := f.createGHStorage(t)
				g.Edge(0, 2).SetDistance(9).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(2, 1).SetDistance(2).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(2, 3).SetDistance(11).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(3, 4).SetDistance(6).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(4, 1).SetDistance(9).SetDecimalBothDir(f.speedEnc, 60, 60)
				p := f.calcPath(g, 0, 4)
				assertDistEquals(t, 20, p.Distance, 1e-4, p.String())
				assertNodesEqual(t, []int{0, 2, 1, 4}, p)
			})

			t.Run("NoPathFound", func(t *testing.T) {
				g := f.createGHStorage(t)
				g.Edge(100, 101)
				p := f.calcPath(g, 0, 1)
				if p.Found {
					t.Fatal("expected no path found for disconnected nodes")
				}

				g2 := f.createGHStorage(t)
				g2.Edge(100, 101)
				g2.Edge(0, 1).SetDistance(7).SetDecimalBothDir(f.speedEnc, 60, 60)
				g2.Edge(5, 6).SetDistance(2).SetDecimalBothDir(f.speedEnc, 60, 60)
				g2.Edge(5, 7).SetDistance(1).SetDecimalBothDir(f.speedEnc, 60, 60)
				g2.Edge(5, 8).SetDistance(1).SetDecimalBothDir(f.speedEnc, 60, 60)
				g2.Edge(7, 8).SetDistance(1).SetDecimalBothDir(f.speedEnc, 60, 60)
				p2 := f.calcPath(g2, 0, 5)
				if p2.Found {
					t.Fatal("expected no path found between disconnected areas")
				}

				g3 := f.createGHStorage(t)
				g3.Edge(0, 1).SetDistance(1).SetDecimalBothDir(f.speedEnc, 60, 0)
				g3.Edge(0, 2).SetDistance(1).SetDecimalBothDir(f.speedEnc, 60, 60)
				p3 := f.calcPathWithWeighting(g3, f.defaultWeighting, f.defaultMaxVisited, 1, 2)
				if p3.Found {
					t.Fatal("expected no path from 1 to 2 (one-way)")
				}
				p4 := f.calcPathWithWeighting(g3, f.defaultWeighting, f.defaultMaxVisited, 2, 1)
				if !p4.Found {
					t.Fatal("expected path from 2 to 1")
				}
			})

			t.Run("WikipediaShortestPath", func(t *testing.T) {
				g := f.createGHStorage(t)
				initWikipediaTestGraph(g, f.speedEnc)
				p := f.calcPath(g, 0, 4)
				assertNodesEqual(t, []int{0, 2, 5, 4}, p, p.String())
				assertDistEquals(t, 20, p.Distance, 1e-4, p.String())
			})

			t.Run("CalcIf1EdgeAway", func(t *testing.T) {
				g := f.createGHStorage(t)
				initTestStorage(g, f.speedEnc)
				p := f.calcPath(g, 1, 2)
				assertNodesEqual(t, []int{1, 2}, p)
				assertDistEquals(t, 35.1, p.Distance, 0.1, p.String())
			})

			t.Run("MaxVisitedNodes", func(t *testing.T) {
				g := f.createGHStorage(t)
				initBiGraph(g, f.speedEnc)
				p := f.calcPath(g, 0, 4)
				if !p.Found {
					t.Fatal("expected path found with unlimited visited nodes")
				}
				p2 := f.calcPathWithWeighting(g, f.defaultWeighting, 3, 0, 4)
				if p2.Found {
					t.Fatal("expected no path found with maxVisitedNodes=3")
				}
			})

			t.Run("Bidirectional2", func(t *testing.T) {
				g := f.createGHStorage(t)
				g.Edge(0, 1).SetDistance(100).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(1, 2).SetDistance(1).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(2, 3).SetDistance(1).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(3, 4).SetDistance(1).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(4, 5).SetDistance(20).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(5, 6).SetDistance(10).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(6, 7).SetDistance(5).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(7, 0).SetDistance(5).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(3, 8).SetDistance(20).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(8, 6).SetDistance(20).SetDecimalBothDir(f.speedEnc, 60, 60)
				p := f.calcPath(g, 0, 4)
				assertDistEquals(t, 40, p.Distance, 1e-4, p.String())
				if len(p.CalcNodes()) != 5 {
					t.Fatalf("expected 5 nodes, got %d: %v", len(p.CalcNodes()), p.CalcNodes())
				}
				assertNodesEqual(t, []int{0, 7, 6, 5, 4}, p)
			})

			t.Run("CalcIfEmptyWay", func(t *testing.T) {
				g := f.createGHStorage(t)
				initTestStorage(g, f.speedEnc)
				p := f.calcPath(g, 0, 0)
				assertPathFromEqualsTo(t, p, 0)
			})

			t.Run("CreateAlgoTwice", func(t *testing.T) {
				g := f.createGHStorage(t)
				g.Edge(0, 1).SetDistance(1).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(1, 2).SetDistance(1).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(2, 3).SetDistance(1).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(3, 4).SetDistance(1).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(4, 5).SetDistance(1).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(5, 6).SetDistance(1).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(6, 7).SetDistance(1).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(7, 0).SetDistance(1).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(3, 8).SetDistance(1).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(8, 6).SetDistance(1).SetDecimalBothDir(f.speedEnc, 60, 60)

				p1 := f.calcPath(g, 0, 4)
				p2 := f.calcPath(g, 0, 4)
				nodes1 := p1.CalcNodes()
				nodes2 := p2.CalcNodes()
				if len(nodes1) != len(nodes2) {
					t.Fatalf("two runs produced different node counts: %v vs %v", nodes1, nodes2)
				}
				for i := range nodes1 {
					if nodes1[i] != nodes2[i] {
						t.Fatalf("two runs produced different nodes at %d: %v vs %v", i, nodes1, nodes2)
					}
				}
			})

			t.Run("BidirectionalLinear", func(t *testing.T) {
				g := f.createGHStorage(t)
				g.Edge(2, 1).SetDistance(2).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(2, 3).SetDistance(11).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(5, 4).SetDistance(6).SetDecimalBothDir(f.speedEnc, 60, 60)
				g.Edge(4, 1).SetDistance(9).SetDecimalBothDir(f.speedEnc, 60, 60)
				p := f.calcPath(g, 3, 5)
				assertDistEquals(t, 28, p.Distance, 1e-4, p.String())
				assertNodesEqual(t, []int{3, 2, 1, 4, 5}, p)
			})

			t.Run("Bidirectional", func(t *testing.T) {
				g := f.createGHStorage(t)
				initBiGraph(g, f.speedEnc)

				p := f.calcPath(g, 0, 4)
				assertNodesEqual(t, []int{0, 7, 6, 8, 3, 4}, p, p.String())
				assertDistEquals(t, 335.8, p.Distance, 0.1, p.String())

				p2 := f.calcPath(g, 1, 2)
				assertNodesEqual(t, []int{1, 2}, p2, p2.String())
				assertDistEquals(t, 10007.7, p2.Distance, 0.1, p2.String())
			})
		})
	}
}

func TestDijkstraBidirectionRef_GetName(t *testing.T) {
	f := newBidirFixture("name_test", routingutil.NodeBased)
	g := f.createGHStorage(t)
	algo := NewDijkstraBidirectionRef(g, f.defaultWeighting, f.traversalMode)
	if algo.GetName() != AlgoDijkstraBi {
		t.Fatalf("expected name %q, got %q", AlgoDijkstraBi, algo.GetName())
	}
}

func TestDijkstraBidirectionRef_CalcPaths(t *testing.T) {
	f := newBidirFixture("calcpaths_test", routingutil.NodeBased)
	g := f.createGHStorage(t)
	g.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 60, 60)

	algo := NewDijkstraBidirectionRef(g, f.defaultWeighting, f.traversalMode)
	paths := algo.CalcPaths(0, 1)
	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}
	if !paths[0].Found {
		t.Fatal("expected path to be found")
	}
}

func TestDijkstraBidirectionRef_PanicsOnSecondRun(t *testing.T) {
	f := newBidirFixture("panic_test", routingutil.NodeBased)
	g := f.createGHStorage(t)
	g.Edge(0, 1).SetDistance(1).SetDecimalBothDir(f.speedEnc, 60, 60)

	algo := NewDijkstraBidirectionRef(g, f.defaultWeighting, f.traversalMode)
	algo.CalcPath(0, 1)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on second CalcPath call")
		}
	}()
	algo.CalcPath(0, 1)
}
