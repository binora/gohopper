package routing

import (
	"math"
	"testing"

	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// --- test fixture ---

// dijkstraFixture holds the shared state for parameterised Dijkstra tests,
// mirroring Java RoutingAlgorithmTest.Fixture.
type dijkstraFixture struct {
	name              string
	traversalMode     routingutil.TraversalMode
	speedEnc          ev.DecimalEncodedValue
	defaultWeighting  weighting.Weighting
	defaultMaxVisited int
	bytesForFlags     int
}

func newDijkstraFixture(name string, tMode routingutil.TraversalMode) *dijkstraFixture {
	speedEnc := ev.NewDecimalEncodedValueImpl("car_speed", 5, 5, true)
	cfg := ev.NewInitializerConfig()
	speedEnc.Init(cfg)
	return &dijkstraFixture{
		name:              name,
		traversalMode:     tMode,
		speedEnc:          speedEnc,
		defaultWeighting:  weighting.NewSpeedWeighting(speedEnc),
		defaultMaxVisited: math.MaxInt,
		bytesForFlags:     cfg.GetRequiredBytes(),
	}
}

func (f *dijkstraFixture) createGHStorage(t *testing.T) *storage.BaseGraph {
	t.Helper()
	b := storage.NewBaseGraphBuilder(f.bytesForFlags)
	if f.traversalMode.IsEdgeBased() {
		b.SetWithTurnCosts(true)
	}
	g := b.CreateGraph()
	t.Cleanup(func() { g.Close() })
	return g
}

func (f *dijkstraFixture) calcPath(g *storage.BaseGraph, from, to int) *Path {
	return f.calcPathWithWeighting(g, f.defaultWeighting, f.defaultMaxVisited, from, to)
}

func (f *dijkstraFixture) calcPathWithWeighting(g *storage.BaseGraph, w weighting.Weighting, maxVisitedNodes int, from, to int) *Path {
	algo := NewDijkstra(g, w, f.traversalMode)
	algo.SetMaxVisitedNodes(maxVisitedNodes)
	return algo.CalcPath(from, to)
}

// --- graph initialisation helpers (ported from Java RoutingAlgorithmTest) ---

// initTestStorage builds the test-graph.svg graph.
//
//	0-1-2-3
//	|/|\|  |
//	4-5-6  |
//	  |   /
//	  7--/
func initTestStorage(g storage.Graph, speedEnc ev.DecimalEncodedValue) {
	g.Edge(0, 1).SetDistance(7).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(0, 4).SetDistance(6).SetDecimalBothDir(speedEnc, 60, 60)

	g.Edge(1, 4).SetDistance(2).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(1, 5).SetDistance(8).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(1, 2).SetDistance(2).SetDecimalBothDir(speedEnc, 60, 60)

	g.Edge(2, 5).SetDistance(5).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(2, 3).SetDistance(2).SetDecimalBothDir(speedEnc, 60, 60)

	g.Edge(3, 5).SetDistance(2).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(3, 7).SetDistance(10).SetDecimalBothDir(speedEnc, 60, 60)

	g.Edge(4, 6).SetDistance(4).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(4, 5).SetDistance(7).SetDecimalBothDir(speedEnc, 60, 60)

	g.Edge(5, 6).SetDistance(2).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(5, 7).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)

	edge67 := g.Edge(6, 7).SetDistance(5).SetDecimalBothDir(speedEnc, 60, 60)

	updateDistancesFor(g, 0, 0.0010, 0.00001)
	updateDistancesFor(g, 1, 0.0008, 0.0000)
	updateDistancesFor(g, 2, 0.0005, 0.0001)
	updateDistancesFor(g, 3, 0.0006, 0.0002)
	updateDistancesFor(g, 4, 0.0009, 0.0001)
	updateDistancesFor(g, 5, 0.0007, 0.0001)
	updateDistancesFor(g, 6, 0.0009, 0.0002)
	updateDistancesFor(g, 7, 0.0008, 0.0003)

	edge67.SetDistance(5 * edge67.GetDistance())
}

// initBiGraph builds the bidirectional test graph.
//
//	0-1-2-3-4
//	|     / |
//	|    8  |
//	\   /   |
//	 7-6----5
func initBiGraph(g storage.Graph, speedEnc ev.DecimalEncodedValue) {
	g.Edge(0, 1).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(1, 2).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(2, 3).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(3, 4).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(4, 5).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(5, 6).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(6, 7).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(7, 0).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(3, 8).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(8, 6).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)

	updateDistancesFor(g, 0, 0.001, 0)
	updateDistancesFor(g, 1, 0.100, 0.0005)
	updateDistancesFor(g, 2, 0.010, 0.0010)
	updateDistancesFor(g, 3, 0.001, 0.0011)
	updateDistancesFor(g, 4, 0.001, 0.00111)

	updateDistancesFor(g, 8, 0.0005, 0.0011)

	updateDistancesFor(g, 7, 0, 0)
	updateDistancesFor(g, 6, 0, 0.001)
	updateDistancesFor(g, 5, 0, 0.004)
}

// initWikipediaTestGraph builds the Wikipedia Dijkstra example graph.
func initWikipediaTestGraph(g storage.Graph, speedEnc ev.DecimalEncodedValue) {
	g.Edge(0, 1).SetDistance(7).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(0, 2).SetDistance(9).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(0, 5).SetDistance(14).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(1, 2).SetDistance(10).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(1, 3).SetDistance(15).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(2, 5).SetDistance(2).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(2, 3).SetDistance(11).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(3, 4).SetDistance(6).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(4, 5).SetDistance(9).SetDecimalBothDir(speedEnc, 60, 60)
}

// updateDistancesFor sets the node location and recalculates all edge distances from the node.
// Mirrors Java GHUtility.updateDistancesFor.
func updateDistancesFor(g storage.Graph, node int, lat, lon float64) {
	na := g.GetNodeAccess()
	na.SetNode(node, lat, lon, 0)
	iter := g.CreateEdgeExplorer(routingutil.AllEdges).SetBaseNode(node)
	for iter.Next() {
		pl := iter.FetchWayGeometry(util.FetchModeAll)
		iter.SetDistance(util.DistEarth.CalcPointListDistance(pl))
	}
}

// --- assertion helpers ---

func assertNodesEqual(t *testing.T, expected []int, path *Path, msgAndArgs ...interface{}) {
	t.Helper()
	nodes := path.CalcNodes()
	if len(nodes) != len(expected) {
		t.Fatalf("expected %d nodes %v, got %d nodes %v. path: %s %v",
			len(expected), expected, len(nodes), nodes, path.String(), msgAndArgs)
	}
	for i, n := range expected {
		if nodes[i] != n {
			t.Fatalf("node mismatch at index %d: expected %d, got %d. full: %v vs %v. path: %s %v",
				i, n, nodes[i], expected, nodes, path.String(), msgAndArgs)
		}
	}
}

func assertDistEquals(t *testing.T, expected, actual, delta float64, msgAndArgs ...interface{}) {
	t.Helper()
	if math.Abs(expected-actual) > delta {
		t.Fatalf("distance mismatch: expected %v, got %v (delta %v) %v",
			expected, actual, delta, msgAndArgs)
	}
}

func assertPathFromEqualsTo(t *testing.T, p *Path, node int) {
	t.Helper()
	if !p.Found {
		t.Fatal("expected path to be found")
	}
	assertNodesEqual(t, []int{node}, p)
	points := p.CalcPoints()
	if points.Size() != 1 {
		t.Fatalf("expected 1 point, got %d", points.Size())
	}
	edges := p.CalcEdges()
	if len(edges) != 0 {
		t.Fatalf("expected 0 edges, got %d", len(edges))
	}
	if math.Abs(p.Weight) > 1e-4 {
		t.Fatalf("expected weight ~0, got %v", p.Weight)
	}
}

// --- tests ---

func TestDijkstra(t *testing.T) {
	fixtures := []*dijkstraFixture{
		newDijkstraFixture("node_based", routingutil.NodeBased),
		newDijkstraFixture("edge_based", routingutil.EdgeBased),
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
				// 0--2--1
				//    |  |
				//    3--4
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
				// isolated nodes
				g := f.createGHStorage(t)
				g.Edge(100, 101)
				p := f.calcPath(g, 0, 1)
				if p.Found {
					t.Fatal("expected no path found for disconnected nodes")
				}

				// two disconnected areas
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

				// disconnected as directed graph: 2-0->1
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
				// unlimited
				p := f.calcPath(g, 0, 4)
				if !p.Found {
					t.Fatal("expected path found with unlimited visited nodes")
				}
				// limited to 3
				p2 := f.calcPathWithWeighting(g, f.defaultWeighting, 3, 0, 4)
				if p2.Found {
					t.Fatal("expected no path found with maxVisitedNodes=3")
				}
			})

			t.Run("Bidirectional2", func(t *testing.T) {
				// 0-1-2-3-4
				// |     / |
				// |    8  |
				// \   /   /
				//  7-6-5-/
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
				// 0-1-2-3-4
				// |     / |
				// |    8  |
				// \   /   /
				//  7-6-5-/
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
				// 3--2--1--4--5
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

// TestDijkstra_GetName verifies the algorithm name.
func TestDijkstra_GetName(t *testing.T) {
	f := newDijkstraFixture("name_test", routingutil.NodeBased)
	g := f.createGHStorage(t)
	algo := NewDijkstra(g, f.defaultWeighting, f.traversalMode)
	if algo.GetName() != AlgoDijkstra {
		t.Fatalf("expected name %q, got %q", AlgoDijkstra, algo.GetName())
	}
}

// TestDijkstra_CalcPaths verifies CalcPaths returns a single-element slice.
func TestDijkstra_CalcPaths(t *testing.T) {
	f := newDijkstraFixture("calcpaths_test", routingutil.NodeBased)
	g := f.createGHStorage(t)
	g.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 60, 60)

	algo := NewDijkstra(g, f.defaultWeighting, f.traversalMode)
	paths := algo.CalcPaths(0, 1)
	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}
	if !paths[0].Found {
		t.Fatal("expected path to be found")
	}
}

// TestDijkstra_PanicsOnSecondRun verifies the algorithm panics if used twice.
func TestDijkstra_PanicsOnSecondRun(t *testing.T) {
	f := newDijkstraFixture("panic_test", routingutil.NodeBased)
	g := f.createGHStorage(t)
	g.Edge(0, 1).SetDistance(1).SetDecimalBothDir(f.speedEnc, 60, 60)

	algo := NewDijkstra(g, f.defaultWeighting, f.traversalMode)
	algo.CalcPath(0, 1)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on second CalcPath call")
		}
	}()
	algo.CalcPath(0, 1)
}
