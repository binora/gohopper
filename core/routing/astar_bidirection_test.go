package routing

import (
	"testing"

	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
)

// astarBidirFixture reuses bidirFixture and overrides calcPath to use AStarBidirection.
type astarBidirFixture struct {
	*bidirFixture
}

func newAStarBidirFixture(name string, tMode routingutil.TraversalMode) *astarBidirFixture {
	return &astarBidirFixture{bidirFixture: newBidirFixture(name, tMode)}
}

func (f *astarBidirFixture) calcPath(g *storage.BaseGraph, from, to int) *Path {
	return f.calcPathWithWeighting(g, f.defaultWeighting, f.defaultMaxVisited, from, to)
}

func (f *astarBidirFixture) calcPathWithWeighting(g *storage.BaseGraph, w weighting.Weighting, maxVisitedNodes int, from, to int) *Path {
	algo := NewAStarBidirection(g, w, f.traversalMode)
	algo.SetMaxVisitedNodes(maxVisitedNodes)
	return algo.CalcPath(from, to)
}

func TestAStarBidirection(t *testing.T) {
	fixtures := []*astarBidirFixture{
		newAStarBidirFixture("node_based", routingutil.NodeBased),
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
			})

			t.Run("WikipediaShortestPath", func(t *testing.T) {
				g := f.createGHStorage(t)
				initWikipediaTestGraph(g, f.speedEnc)
				p := f.calcPath(g, 0, 4)
				assertNodesEqual(t, []int{0, 2, 5, 4}, p, p.String())
				assertDistEquals(t, 20, p.Distance, 1e-4, p.String())
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

			t.Run("Bidirectional", func(t *testing.T) {
				g := f.createGHStorage(t)
				initBiGraph(g, f.speedEnc)
				p := f.calcPath(g, 0, 4)
				assertNodesEqual(t, []int{0, 7, 6, 8, 3, 4}, p, p.String())
				assertDistEquals(t, 335.8, p.Distance, 0.1, p.String())
			})

			t.Run("AStarBidirProducesSameResultAsDijkstra", func(t *testing.T) {
				g := f.createGHStorage(t)
				initTestStorage(g, f.speedEnc)

				dijkstraPath := NewDijkstra(g, f.defaultWeighting, f.traversalMode).CalcPath(0, 7)
				astarBiPath := f.calcPath(g, 0, 7)

				assertDistEquals(t, dijkstraPath.Distance, astarBiPath.Distance, 1e-6)
				assertDistEquals(t, dijkstraPath.Weight, astarBiPath.Weight, 1e-6)
			})
		})
	}
}

// infeasibleApproximator deliberately violates the admissibility criteria of A*.
type infeasibleApproximator struct {
	to int
}

func (a *infeasibleApproximator) Approximate(currentNode int) float64 {
	if a.to != 9 {
		return 0
	}
	if currentNode == 10 {
		return 1000
	}
	return 0
}

func (a *infeasibleApproximator) SetTo(toNode int) {
	a.to = toNode
}

func (a *infeasibleApproximator) Reverse() weighting.WeightApproximator {
	return &infeasibleApproximator{}
}

func (a *infeasibleApproximator) GetSlack() float64 {
	return 0
}

func TestAStarBidirection_InfeasibleApproximator(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 2, 1, true)
	cfg := ev.NewInitializerConfig()
	speedEnc.Init(cfg)
	bytesForFlags := cfg.GetRequiredBytes()

	g := storage.NewBaseGraphBuilder(bytesForFlags).CreateGraph()
	t.Cleanup(func() { g.Close() })

	// 0-1----2-3-4----5-6-7-8-9
	//    \  /
	//     10
	g.Edge(0, 1).SetDecimal(speedEnc, 1).SetReverseDecimal(speedEnc, 0).SetDistance(100)
	g.Edge(2, 1).SetDistance(300).SetDecimal(speedEnc, 0).SetReverseDecimal(speedEnc, 1)
	g.Edge(2, 3).SetDecimal(speedEnc, 1).SetReverseDecimal(speedEnc, 0).SetDistance(100)
	g.Edge(3, 4).SetDecimal(speedEnc, 1).SetReverseDecimal(speedEnc, 0).SetDistance(100)
	g.Edge(4, 5).SetDecimal(speedEnc, 1).SetReverseDecimal(speedEnc, 0).SetDistance(10_000)
	g.Edge(5, 6).SetDecimal(speedEnc, 1).SetReverseDecimal(speedEnc, 0).SetDistance(100)
	g.Edge(6, 7).SetDecimal(speedEnc, 1).SetReverseDecimal(speedEnc, 0).SetDistance(100)
	g.Edge(7, 8).SetDecimal(speedEnc, 1).SetReverseDecimal(speedEnc, 0).SetDistance(100)
	g.Edge(8, 9).SetDecimal(speedEnc, 1).SetReverseDecimal(speedEnc, 0).SetDistance(100)
	g.Edge(1, 10).SetDecimal(speedEnc, 1).SetReverseDecimal(speedEnc, 0).SetDistance(100)
	g.Edge(10, 2).SetDecimal(speedEnc, 1).SetReverseDecimal(speedEnc, 0).SetDistance(100)

	w := weighting.NewSpeedWeighting(speedEnc)
	algo := NewAStarBidirection(g, w, routingutil.NodeBased)
	algo.SetApproximation(&infeasibleApproximator{})
	path := algo.CalcPath(0, 9)
	// the path is not the shortest path, but the suboptimal one we get for this approximator
	assertDistEquals(t, 11_000, path.Distance, 1e-4, path.String())
	assertNodesEqual(t, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, path)

	// this returns the correct path
	dijkstra := NewDijkstra(g, w, routingutil.NodeBased)
	optimalPath := dijkstra.CalcPath(0, 9)
	assertDistEquals(t, 10_900, optimalPath.Distance, 1e-4, optimalPath.String())
	assertNodesEqual(t, []int{0, 1, 10, 2, 3, 4, 5, 6, 7, 8, 9}, optimalPath)
}

func TestAStarBidirection_GetName(t *testing.T) {
	f := newAStarBidirFixture("name_test", routingutil.NodeBased)
	g := f.createGHStorage(t)
	algo := NewAStarBidirection(g, f.defaultWeighting, f.traversalMode)
	name := algo.GetName()
	if len(name) == 0 {
		t.Fatal("expected non-empty algorithm name")
	}
}
