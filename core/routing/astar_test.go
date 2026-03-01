package routing

import (
	"testing"

	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
)

// astarFixture reuses dijkstraFixture and overrides calcPath to use AStar.
type astarFixture struct {
	*dijkstraFixture
}

func newAStarFixture(name string, tMode routingutil.TraversalMode) *astarFixture {
	return &astarFixture{dijkstraFixture: newDijkstraFixture(name, tMode)}
}

func (f *astarFixture) calcPath(g *storage.BaseGraph, from, to int) *Path {
	return f.calcPathWithWeighting(g, f.defaultWeighting, f.defaultMaxVisited, from, to)
}

func (f *astarFixture) calcPathWithWeighting(g *storage.BaseGraph, w weighting.Weighting, maxVisitedNodes int, from, to int) *Path {
	algo := NewAStar(g, w, f.traversalMode)
	algo.SetMaxVisitedNodes(maxVisitedNodes)
	return algo.CalcPath(from, to)
}

func TestAStar(t *testing.T) {
	fixtures := []*astarFixture{
		newAStarFixture("node_based", routingutil.NodeBased),
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

			t.Run("AStarProducesSameResultAsDijkstra", func(t *testing.T) {
				g := f.createGHStorage(t)
				initTestStorage(g, f.speedEnc)

				dijkstraPath := NewDijkstra(g, f.defaultWeighting, f.traversalMode).CalcPath(0, 7)
				astarPath := f.calcPath(g, 0, 7)

				assertDistEquals(t, dijkstraPath.Distance, astarPath.Distance, 1e-6)
				assertDistEquals(t, dijkstraPath.Weight, astarPath.Weight, 1e-6)
			})
		})
	}
}

func TestAStar_GetName(t *testing.T) {
	f := newAStarFixture("name_test", routingutil.NodeBased)
	g := f.createGHStorage(t)
	algo := NewAStar(g, f.defaultWeighting, f.traversalMode)
	name := algo.GetName()
	if len(name) == 0 {
		t.Fatal("expected non-empty algorithm name")
	}
}
