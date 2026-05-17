package ch

import (
	"math"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"gohopper/core/routing"
	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"
	webapi "gohopper/web-api"
)

type turnCostQueryFixture struct {
	speedEnc    ev.DecimalEncodedValue
	turnCostEnc ev.DecimalEncodedValue
	graph       *storage.BaseGraph
	weighting   *weighting.SpeedWeighting
	chStore     *storage.CHStorage
	chBuilder   *storage.CHStorageBuilder
}

func newTurnCostQueryFixture() *turnCostQueryFixture {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	turnCostEnc := ev.TurnCostCreate("car", 10)
	em := routingutil.Start().
		Add(speedEnc).
		AddTurnCostEncodedValue(turnCostEnc).
		Build()
	graph := storage.NewBaseGraphBuilder(em.BytesForFlags).SetWithTurnCosts(true).CreateGraph()
	return &turnCostQueryFixture{
		speedEnc:    speedEnc,
		turnCostEnc: turnCostEnc,
		graph:       graph,
		weighting:   weighting.NewSpeedWeightingWithTurnCosts(speedEnc, turnCostEnc, graph.GetTurnCostStorage(), graph.GetNodeAccess(), math.Inf(1)),
	}
}

func (f *turnCostQueryFixture) freezeWithIdentityLevels() {
	f.graph.Freeze()
	f.chStore = storage.CHStorageFromGraph(f.graph, "profile", true)
	f.chBuilder = storage.NewCHStorageBuilder(f.chStore)
	f.chBuilder.SetIdentityLevels()
}

func (f *turnCostQueryFixture) edge(from, to int) util.EdgeIteratorState {
	iter := f.graph.CreateEdgeExplorer(routingutil.AllEdges).SetBaseNode(from)
	for iter.Next() {
		if iter.GetAdjNode() == to {
			return iter
		}
	}
	panic("edge not found")
}

func (f *turnCostQueryFixture) setTurnCost(from, via, to int, cost float64) {
	fromEdge := f.edge(from, via).GetEdge()
	toEdge := f.edge(via, to).GetEdge()
	f.setTurnCostEdges(fromEdge, via, toEdge, cost)
}

func (f *turnCostQueryFixture) setTurnCostEdges(fromEdge int, via, toEdge int, cost float64) {
	f.graph.GetTurnCostStorage().SetDecimal(f.graph.GetNodeAccess(), f.turnCostEnc, fromEdge, via, toEdge, cost)
}

func (f *turnCostQueryFixture) setRestriction(from, via, to int) {
	f.setTurnCost(from, via, to, math.Inf(1))
}

func (f *turnCostQueryFixture) setRestrictionEdges(fromEdge int, via, toEdge int) {
	f.setTurnCostEdges(fromEdge, via, toEdge, math.Inf(1))
}

func (f *turnCostQueryFixture) addShortcut(from, to, firstOrigEdgeKey, lastOrigEdgeKey, skipped1, skipped2 int, weight float64, reverse bool) {
	flags := ScFwdDir
	if reverse {
		flags = ScBwdDir
	}
	f.chBuilder.AddShortcutEdgeBased(from, to, flags, weight, skipped1, skipped2, firstOrigEdgeKey, lastOrigEdgeKey)
}

func (f *turnCostQueryFixture) calcPath(algoName string, from, to int) *routing.Path {
	chGraph := storage.NewRoutingCHGraph(f.graph, f.chStore, f.weighting)
	opts := webapi.NewPMap().PutObject(chRoutingAlgorithmKey, algoName)
	return NewCHRoutingAlgorithmFactory(chGraph).CreateAlgo(opts).CalcPath(from, to)
}

func (f *turnCostQueryFixture) assertPath(t *testing.T, algoName string, from, to int, expectedWeight float64, expectedNodes []int, expectedDistance float64, expectedTime int64) {
	t.Helper()
	path := f.calcPath(algoName, from, to)
	assert.True(t, path.Found, "expected path from %d to %d", from, to)
	assert.InDelta(t, expectedWeight, path.Weight, 1e-6)
	assert.InDelta(t, expectedDistance, path.Distance, 1e-6)
	assert.Equal(t, expectedTime, path.Time)
	assert.Equal(t, expectedNodes, path.CalcNodes())
}

func (f *turnCostQueryFixture) assertNotFound(t *testing.T, algoName string, from, to int) {
	t.Helper()
	path := f.calcPath(algoName, from, to)
	assert.False(t, path.Found, "unexpected path from %d to %d", from, to)
}

func TestCHQueryWithTurnCosts_BidirectedNoShortcutsSmallGraph(t *testing.T) {
	for _, algoName := range []string{routing.AlgoAStarBi, routing.AlgoDijkstraBi} {
		t.Run(algoName, func(t *testing.T) {
			f := newTurnCostQueryFixture()
			f.graph.Edge(1, 0).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 10)
			f.graph.Edge(0, 2).SetDistance(50).SetDecimalBothDir(f.speedEnc, 10, 10)
			f.setTurnCost(1, 0, 2, 3)
			f.freezeWithIdentityLevels()

			for i := 0; i < 3; i++ {
				f.assertPath(t, algoName, i, i, 0, []int{i}, 0, 0)
			}
			f.assertPath(t, algoName, 1, 2, 11, []int{1, 0, 2}, 80, 11_000)
			f.assertPath(t, algoName, 2, 1, 8, []int{2, 0, 1}, 80, 8_000)
			f.assertPath(t, algoName, 0, 1, 3, []int{0, 1}, 30, 3_000)
			f.assertPath(t, algoName, 0, 2, 5, []int{0, 2}, 50, 5_000)
			f.assertPath(t, algoName, 1, 0, 3, []int{1, 0}, 30, 3_000)
			f.assertPath(t, algoName, 2, 0, 5, []int{2, 0}, 50, 5_000)
		})
	}
}

func TestCHQueryWithTurnCosts_LoopShortcutBackwardSearch(t *testing.T) {
	for _, algoName := range []string{routing.AlgoAStarBi, routing.AlgoDijkstraBi} {
		t.Run(algoName, func(t *testing.T) {
			f := newTurnCostQueryFixture()
			f.graph.Edge(0, 7).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(7, 8).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(8, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(4, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(1, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(3, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(2, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(4, 6).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(6, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.setRestriction(8, 4, 6)
			f.setRestriction(8, 4, 2)
			f.setRestriction(1, 4, 6)
			f.freezeWithIdentityLevels()

			f.addShortcut(3, 4, 6, 8, 3, 4, 2, true)
			f.addShortcut(3, 4, 10, 12, 5, 6, 2, false)
			f.addShortcut(4, 4, 6, 13, 9, 10, 4, false)
			f.addShortcut(4, 8, 4, 12, 2, 11, 5, true)
			f.addShortcut(6, 8, 4, 14, 12, 7, 6, true)

			f.assertPath(t, algoName, 0, 5, 9, []int{0, 7, 8, 4, 1, 3, 2, 4, 6, 5}, 90, 9_000)
		})
	}
}

func TestCHQueryWithTurnCosts_LoopShortcutForwardSearch(t *testing.T) {
	for _, algoName := range []string{routing.AlgoAStarBi, routing.AlgoDijkstraBi} {
		t.Run(algoName, func(t *testing.T) {
			f := newTurnCostQueryFixture()
			f.graph.Edge(5, 6).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(6, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(4, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(1, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(3, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(2, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(4, 7).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(7, 8).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(8, 0).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.setRestriction(6, 4, 7)
			f.setRestriction(6, 4, 2)
			f.setRestriction(1, 4, 7)
			f.freezeWithIdentityLevels()

			f.addShortcut(3, 4, 4, 6, 2, 3, 2, true)
			f.addShortcut(3, 4, 8, 10, 4, 5, 2, false)
			f.addShortcut(4, 4, 4, 10, 9, 10, 4, false)
			f.addShortcut(4, 6, 3, 10, 1, 11, 5, true)
			f.addShortcut(6, 7, 2, 12, 12, 6, 6, false)

			f.assertPath(t, algoName, 5, 0, 9, []int{5, 6, 4, 1, 3, 2, 4, 7, 8, 0}, 90, 9_000)
		})
	}
}

func TestCHQueryWithTurnCosts_BidirectedNoShortcuts(t *testing.T) {
	for _, algoName := range []string{routing.AlgoAStarBi, routing.AlgoDijkstraBi} {
		t.Run(algoName, func(t *testing.T) {
			f := newTurnCostQueryFixture()
			f.graph.Edge(0, 2).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 10)
			f.graph.Edge(2, 4).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 10)
			f.graph.Edge(4, 6).SetDistance(70).SetDecimalBothDir(f.speedEnc, 10, 10)
			f.graph.Edge(6, 5).SetDistance(90).SetDecimalBothDir(f.speedEnc, 10, 10)
			f.graph.Edge(5, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
			f.graph.Edge(3, 1).SetDistance(40).SetDecimalBothDir(f.speedEnc, 10, 10)
			f.setTurnCost(0, 2, 4, 3)
			f.setTurnCost(4, 6, 5, 6)
			f.setTurnCost(5, 6, 4, 2)
			f.setTurnCost(5, 3, 1, 5)
			f.freezeWithIdentityLevels()

			f.assertPath(t, algoName, 0, 1, 40, []int{0, 2, 4, 6, 5, 3, 1}, 260, 40_000)
			f.assertPath(t, algoName, 1, 0, 28, []int{1, 3, 5, 6, 4, 2, 0}, 260, 28_000)
			f.assertPath(t, algoName, 4, 3, 23, []int{4, 6, 5, 3}, 170, 23_000)
			f.assertPath(t, algoName, 0, 0, 0, []int{0}, 0, 0)
			f.assertPath(t, algoName, 4, 4, 0, []int{4}, 0, 0)
		})
	}
}

func TestCHQueryWithTurnCosts_DirectedSingleShortcut(t *testing.T) {
	for _, algoName := range []string{routing.AlgoAStarBi, routing.AlgoDijkstraBi} {
		t.Run(algoName, func(t *testing.T) {
			f := newTurnCostQueryFixture()
			f.graph.Edge(1, 2).SetDistance(40).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(2, 0).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(0, 3).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(3, 4).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.setTurnCost(1, 2, 0, 5)
			f.setTurnCost(2, 0, 3, 2)
			f.setTurnCost(0, 3, 4, 1)
			f.freezeWithIdentityLevels()
			f.addShortcut(2, 3, 2, 4, 1, 2, 7, false)

			f.assertPath(t, algoName, 1, 4, 19, []int{1, 2, 0, 3, 4}, 110, 19_000)
			f.assertPath(t, algoName, 2, 4, 10, []int{2, 0, 3, 4}, 70, 10_000)
			f.assertPath(t, algoName, 0, 4, 6, []int{0, 3, 4}, 50, 6_000)
			f.assertPath(t, algoName, 1, 0, 11, []int{1, 2, 0}, 60, 11_000)
			f.assertPath(t, algoName, 0, 4, 6, []int{0, 3, 4}, 50, 6_000)
		})
	}
}

func TestCHQueryWithTurnCosts_DirectedSingleShortcutForwardSearchStopsQuickly(t *testing.T) {
	for _, algoName := range []string{routing.AlgoAStarBi, routing.AlgoDijkstraBi} {
		t.Run(algoName, func(t *testing.T) {
			f := newTurnCostQueryFixture()
			f.graph.Edge(1, 2).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(2, 0).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(0, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(3, 4).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.setTurnCost(1, 2, 0, 2)
			f.setTurnCost(0, 3, 4, 4)
			f.freezeWithIdentityLevels()
			f.addShortcut(2, 3, 2, 4, 1, 2, 4, false)

			f.assertPath(t, algoName, 1, 4, 15, []int{1, 2, 0, 3, 4}, 90, 15_000)
		})
	}
}

func TestCHQueryWithTurnCosts_DirectedTwoShortcuts(t *testing.T) {
	for _, algoName := range []string{routing.AlgoAStarBi, routing.AlgoDijkstraBi} {
		t.Run(algoName, func(t *testing.T) {
			f := newTurnCostQueryFixture()
			f.graph.Edge(2, 3).SetDistance(40).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(3, 1).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(1, 0).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(0, 4).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.setTurnCost(2, 3, 1, 5)
			f.setTurnCost(3, 1, 0, 2)
			f.setTurnCost(1, 0, 4, 1)
			f.freezeWithIdentityLevels()
			f.addShortcut(1, 4, 4, 6, 2, 3, 6, false)
			f.addShortcut(3, 4, 2, 6, 1, 4, 10, false)

			f.assertPath(t, algoName, 2, 4, 19, []int{2, 3, 1, 0, 4}, 110, 19_000)
			f.assertPath(t, algoName, 1, 4, 6, []int{1, 0, 4}, 50, 6_000)
			f.assertPath(t, algoName, 2, 0, 16, []int{2, 3, 1, 0}, 90, 16_000)
			f.assertPath(t, algoName, 3, 4, 10, []int{3, 1, 0, 4}, 70, 10_000)
			f.assertPath(t, algoName, 2, 1, 11, []int{2, 3, 1}, 60, 11_000)
		})
	}
}

func TestCHQueryWithTurnCosts_DirectConnectionIsNotBestPath(t *testing.T) {
	for _, algoName := range []string{routing.AlgoAStarBi, routing.AlgoDijkstraBi} {
		t.Run(algoName, func(t *testing.T) {
			f := newTurnCostQueryFixture()
			f.graph.Edge(0, 2).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(2, 3).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(3, 1).SetDistance(90).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(0, 1).SetDistance(500).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.setTurnCost(2, 3, 1, 4)
			f.freezeWithIdentityLevels()

			f.assertPath(t, algoName, 0, 1, 18, []int{0, 2, 3, 1}, 140, 18_000)
		})
	}
}

func TestCHQueryWithTurnCosts_UpwardSearchRunsIntoTarget(t *testing.T) {
	for _, algoName := range []string{routing.AlgoAStarBi, routing.AlgoDijkstraBi} {
		t.Run(algoName, func(t *testing.T) {
			f := newTurnCostQueryFixture()
			f.graph.Edge(0, 1).SetDistance(90).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(1, 5).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(1, 3).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(3, 4).SetDistance(40).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(5, 4).SetDistance(60).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(4, 2).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.setTurnCost(1, 3, 4, 3)
			f.freezeWithIdentityLevels()

			f.assertPath(t, algoName, 0, 4, 17, []int{0, 1, 5, 4}, 170, 17_000)
		})
	}
}

func TestCHQueryWithTurnCosts_DownwardSearchRunsIntoTarget(t *testing.T) {
	for _, algoName := range []string{routing.AlgoAStarBi, routing.AlgoDijkstraBi} {
		t.Run(algoName, func(t *testing.T) {
			f := newTurnCostQueryFixture()
			f.graph.Edge(1, 0).SetDistance(90).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(2, 0).SetDistance(140).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(2, 1).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(3, 2).SetDistance(90).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.freezeWithIdentityLevels()

			f.assertPath(t, algoName, 3, 0, 20, []int{3, 2, 1, 0}, 200, 20_000)
		})
	}
}

func TestCHQueryWithTurnCosts_IncomingShortcut(t *testing.T) {
	for _, algoName := range []string{routing.AlgoAStarBi, routing.AlgoDijkstraBi} {
		t.Run(algoName, func(t *testing.T) {
			f := newTurnCostQueryFixture()
			f.graph.Edge(0, 1).SetDistance(90).SetDecimalBothDir(f.speedEnc, 10, 10)
			f.graph.Edge(0, 3).SetDistance(140).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(3, 2).SetDistance(90).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.freezeWithIdentityLevels()
			f.addShortcut(1, 3, 1, 2, 0, 1, 23, false)

			f.assertPath(t, algoName, 0, 2, 23, []int{0, 3, 2}, 230, 23_000)
		})
	}
}

func TestCHQueryWithTurnCosts_FwdBwdSearchesMeetWithUTurn(t *testing.T) {
	for _, algoName := range []string{routing.AlgoAStarBi, routing.AlgoDijkstraBi} {
		t.Run(algoName, func(t *testing.T) {
			f := newTurnCostQueryFixture()
			f.graph.Edge(0, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(2, 3).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 10)
			f.graph.Edge(2, 1).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.setRestriction(0, 2, 1)
			f.setTurnCost(0, 2, 3, 5)
			f.setTurnCost(2, 3, 2, 4)
			f.setTurnCost(3, 2, 1, 7)
			f.freezeWithIdentityLevels()

			f.assertNotFound(t, algoName, 0, 1)
		})
	}
}

func TestCHQueryWithTurnCosts_DoNotMakeUTurn(t *testing.T) {
	for _, algoName := range []string{routing.AlgoAStarBi, routing.AlgoDijkstraBi} {
		t.Run(algoName, func(t *testing.T) {
			checkUTurnNotBeingUsed(t, algoName, false)
		})
	}
}

func TestCHQueryWithTurnCosts_DoNotMakeUTurnToLowerLevelNode(t *testing.T) {
	for _, algoName := range []string{routing.AlgoAStarBi, routing.AlgoDijkstraBi} {
		t.Run(algoName, func(t *testing.T) {
			checkUTurnNotBeingUsed(t, algoName, true)
		})
	}
}

func checkUTurnNotBeingUsed(t *testing.T, algoName string, toLowerLevelNode bool) {
	t.Helper()
	f := newTurnCostQueryFixture()
	nodeA := 4
	nodeB := 5
	if toLowerLevelNode {
		nodeA, nodeB = nodeB, nodeA
	}
	f.graph.Edge(1, nodeA).SetDistance(40).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(0, 3).SetDistance(40).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(nodeB, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	e3toB := f.graph.Edge(3, nodeB).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	e3toA := f.graph.Edge(3, nodeA).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.freezeWithIdentityLevels()
	f.setRestriction(0, 3, nodeB)

	if toLowerLevelNode {
		f.addShortcut(nodeB, nodeA, e3toA.GetReverseEdgeKey(), e3toB.GetEdgeKey(), e3toA.GetEdge(), e3toB.GetEdge(), 2, true)
	} else {
		f.addShortcut(nodeA, nodeB, e3toA.GetReverseEdgeKey(), e3toB.GetEdgeKey(), e3toA.GetEdge(), e3toB.GetEdge(), 2, false)
	}
	f.assertNotFound(t, algoName, 0, 2)
}

func TestCHQueryWithTurnCosts_Loop(t *testing.T) {
	for _, algoName := range []string{routing.AlgoAStarBi, routing.AlgoDijkstraBi} {
		t.Run(algoName, func(t *testing.T) {
			f := newTurnCostQueryFixture()
			edge1 := f.graph.Edge(0, 2).SetDistance(40).SetDecimalBothDir(f.speedEnc, 10, 0)
			edge2 := f.graph.Edge(2, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
			f.graph.Edge(3, 2).SetDistance(70).SetDecimalBothDir(f.speedEnc, 10, 0)
			edge4 := f.graph.Edge(2, 1).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.setRestrictionEdges(edge1.GetEdge(), 2, edge4.GetEdge())
			f.setTurnCostEdges(edge1.GetEdge(), 2, edge2.GetEdge(), 3)
			f.freezeWithIdentityLevels()

			f.assertPath(t, algoName, 0, 1, 18, []int{0, 2, 3, 2, 1}, 150, 18_000)
			f.assertPath(t, algoName, 3, 1, 4, []int{3, 2, 1}, 40, 4_000)
		})
	}
}

func TestCHQueryWithTurnCosts_MultipleBridgeNodes(t *testing.T) {
	for _, algoName := range []string{routing.AlgoAStarBi, routing.AlgoDijkstraBi} {
		t.Run(algoName, func(t *testing.T) {
			f := newTurnCostQueryFixture()
			f.graph.Edge(0, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(0, 3).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(0, 4).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(2, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(3, 1).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(4, 1).SetDistance(60).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.setTurnCost(0, 2, 1, 9)
			f.setTurnCost(0, 3, 1, 2)
			f.setTurnCost(0, 4, 1, 1)
			f.freezeWithIdentityLevels()

			f.assertPath(t, algoName, 0, 1, 7, []int{0, 3, 1}, 50, 7_000)
		})
	}
}

func TestCHQueryWithTurnCosts_ShortcutLoopRecognizedAsIncomingEdge(t *testing.T) {
	for _, algoName := range []string{routing.AlgoAStarBi, routing.AlgoDijkstraBi} {
		t.Run(algoName, func(t *testing.T) {
			f := newTurnCostQueryFixture()
			f.graph.Edge(3, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
			edge1 := f.graph.Edge(4, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
			edge2 := f.graph.Edge(2, 0).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			edge3 := f.graph.Edge(0, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			edge4 := f.graph.Edge(2, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.setRestrictionEdges(edge1.GetEdge(), 2, edge4.GetEdge())
			f.freezeWithIdentityLevels()
			f.addShortcut(2, 2, edge2.GetEdgeKey(), edge3.GetEdgeKey(), edge2.GetEdge(), edge3.GetEdge(), 2, false)

			f.assertPath(t, algoName, 3, 1, 5, []int{3, 4, 2, 0, 2, 1}, 50, 5_000)
		})
	}
}

func TestCHQueryWithTurnCosts_TurnRestrictionSingleLoop(t *testing.T) {
	for _, algoName := range []string{routing.AlgoAStarBi, routing.AlgoDijkstraBi} {
		t.Run(algoName, func(t *testing.T) {
			f := newTurnCostQueryFixture()
			f.graph.Edge(3, 4).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(4, 0).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
			f.graph.Edge(0, 1).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(4, 1).SetDistance(50).SetDecimalBothDir(f.speedEnc, 10, 10)
			f.graph.Edge(4, 2).SetDistance(40).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.setRestriction(3, 4, 2)
			f.freezeWithIdentityLevels()
			f.addShortcut(1, 4, 2, 4, 1, 2, 4, true)
			f.addShortcut(4, 4, 2, 6, 5, 3, 9, false)

			f.assertPath(t, algoName, 3, 2, 15, []int{3, 4, 0, 1, 4, 2}, 150, 15_000)
		})
	}
}

func TestCHQueryWithTurnCosts_SingleLoopInForwardSearch(t *testing.T) {
	for _, algoName := range []string{routing.AlgoAStarBi, routing.AlgoDijkstraBi} {
		t.Run(algoName, func(t *testing.T) {
			runTestWithSingleLoop(t, algoName, true)
		})
	}
}

func TestCHQueryWithTurnCosts_SingleLoopInBackwardSearch(t *testing.T) {
	for _, algoName := range []string{routing.AlgoAStarBi, routing.AlgoDijkstraBi} {
		t.Run(algoName, func(t *testing.T) {
			runTestWithSingleLoop(t, algoName, false)
		})
	}
}

func runTestWithSingleLoop(t *testing.T, algoName string, loopInFwdSearch bool) {
	t.Helper()
	f := newTurnCostQueryFixture()
	nodeA := 0
	nodeB := 6
	if !loopInFwdSearch {
		nodeA, nodeB = nodeB, nodeA
	}
	f.graph.Edge(4, nodeA).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(nodeA, 5).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(5, 2).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 1).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(5, nodeB).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(nodeB, 7).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.setRestriction(nodeA, 5, nodeB)
	f.freezeWithIdentityLevels()
	f.addShortcut(3, 5, 8, 10, 4, 5, 3, false)
	f.addShortcut(3, 5, 4, 6, 2, 3, 3, true)
	f.addShortcut(5, 5, 4, 10, 9, 8, 6, false)

	f.assertPath(t, algoName, 4, 7, 12, []int{4, nodeA, 5, 2, 3, 1, 5, nodeB, 7}, 120, 12_000)
}

func TestCHQueryWithTurnCosts_TurnRestrictionDoubleLoop(t *testing.T) {
	for _, algoName := range []string{routing.AlgoAStarBi, routing.AlgoDijkstraBi} {
		t.Run(algoName, func(t *testing.T) {
			f := newTurnCostQueryFixture()
			f.graph.Edge(0, 1).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 10)
			e1to6 := f.graph.Edge(1, 6).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
			e0to6 := f.graph.Edge(0, 6).SetDistance(40).SetDecimalBothDir(f.speedEnc, 10, 10)
			e2to6 := f.graph.Edge(2, 6).SetDistance(50).SetDecimalBothDir(f.speedEnc, 10, 10)
			f.graph.Edge(2, 3).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 10)
			e3to6 := f.graph.Edge(3, 6).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 10)
			e6to7 := f.graph.Edge(7, 6).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
			e4to7 := f.graph.Edge(7, 4).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 10)
			e5to7 := f.graph.Edge(7, 5).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 10)

			f.setRestrictionEdges(e6to7.GetEdge(), 6, e1to6.GetEdge())
			f.setRestrictionEdges(e6to7.GetEdge(), 6, e2to6.GetEdge())
			f.setRestrictionEdges(e6to7.GetEdge(), 6, e3to6.GetEdge())
			f.setRestrictionEdges(e1to6.GetEdge(), 6, e3to6.GetEdge())
			f.setRestrictionEdges(e1to6.GetEdge(), 6, e6to7.GetEdge())
			f.setRestrictionEdges(e1to6.GetEdge(), 6, e0to6.GetEdge())
			f.setRestrictionEdges(e4to7.GetEdge(), 7, e5to7.GetEdge())
			f.setRestrictionEdges(e5to7.GetEdge(), 7, e4to7.GetEdge())
			f.freezeWithIdentityLevels()

			f.addShortcut(1, 6, 4, 0, 2, 0, 6, true)
			f.addShortcut(3, 6, 6, 8, 3, 4, 8, true)
			f.addShortcut(6, 6, 4, 2, 9, 1, 7, false)
			f.addShortcut(6, 6, 6, 10, 10, 5, 10, false)
			f.addShortcut(6, 7, 12, 2, 6, 11, 8, true)
			f.addShortcut(6, 7, 12, 10, 13, 12, 18, true)
			f.addShortcut(7, 7, 12, 12, 14, 6, 19, false)

			f.assertPath(t, algoName, 4, 5, 24, []int{4, 7, 6, 0, 1, 6, 2, 3, 6, 7, 5}, 240, 24_000)
			f.assertPath(t, algoName, 5, 4, 24, []int{5, 7, 6, 0, 1, 6, 2, 3, 6, 7, 4}, 240, 24_000)
		})
	}
}

func TestCHQueryWithTurnCosts_TurnRestrictionTwoDifferentLoops(t *testing.T) {
	for _, algoName := range []string{routing.AlgoAStarBi, routing.AlgoDijkstraBi} {
		t.Run(algoName, func(t *testing.T) {
			f := newTurnCostQueryFixture()
			f.graph.Edge(0, 1).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(1, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
			f.graph.Edge(5, 0).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(5, 4).SetDistance(50).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(5, 6).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 10)
			f.graph.Edge(6, 4).SetDistance(40).SetDecimalBothDir(f.speedEnc, 10, 10)
			f.graph.Edge(3, 6).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(6, 2).SetDistance(40).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.setRestriction(3, 6, 2)
			f.freezeWithIdentityLevels()

			f.addShortcut(1, 5, 4, 0, 2, 0, 3, true)
			f.addShortcut(5, 5, 4, 2, 8, 1, 4, false)
			f.addShortcut(5, 6, 6, 11, 3, 5, 9, false)
			f.addShortcut(5, 6, 9, 2, 4, 9, 7, true)
			f.addShortcut(6, 6, 9, 8, 11, 4, 10, false)

			distMatrix := [][]int{
				{0, 2, 10, -1, 8, 3, 6},
				{2, 0, 8, -1, 6, 1, 4},
				{-1, -1, 0, -1, -1, -1, -1},
				{7, 7, 17, 0, 7, 6, 3},
				{8, 8, 8, -1, 0, 7, 4},
				{1, 1, 7, -1, 5, 0, 3},
				{4, 4, 4, -1, 4, 3, 0},
			}
			for from, row := range distMatrix {
				for to, weight := range row {
					if weight < 0 {
						f.assertNotFound(t, algoName, from, to)
						continue
					}
					path := f.calcPath(algoName, from, to)
					assert.True(t, path.Found, "expected path from %d to %d", from, to)
					assert.InDelta(t, float64(weight), path.Weight, 1e-6, "unexpected weight from %d to %d", from, to)
					assert.InDelta(t, float64(weight*10), path.Distance, 1e-6, "unexpected distance from %d to %d", from, to)
					assert.Equal(t, int64(weight*1000), path.Time, "unexpected time from %d to %d", from, to)
				}
			}
		})
	}
}

// preparedCHTurnCostFixture mirrors the Java CHTurnCostTest setup. It drives the real
// PrepareContractionHierarchies (rather than hand-building shortcuts) and exposes helpers
// that match Java's prepareCH/automaticPrepareCH/checkPath/compareCHQueryWithDijkstra.
type preparedCHTurnCostFixture struct {
	maxCost     int
	speedEnc    ev.DecimalEncodedValue
	turnCostEnc ev.DecimalEncodedValue
	graph       *storage.BaseGraph
	chConfigs   []*CHConfig
	chConfig    *CHConfig
	chGraph     storage.RoutingCHGraph
	checkStrict bool
}

func newPreparedCHTurnCostFixture() *preparedCHTurnCostFixture {
	maxCost := 10
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	turnCostEnc := ev.TurnCostCreate("car", maxCost)
	em := routingutil.Start().
		Add(speedEnc).
		AddTurnCostEncodedValue(turnCostEnc).
		Build()
	graph := storage.NewBaseGraphBuilder(em.BytesForFlags).SetWithTurnCosts(true).CreateGraph()
	f := &preparedCHTurnCostFixture{
		maxCost:     maxCost,
		speedEnc:    speedEnc,
		turnCostEnc: turnCostEnc,
		graph:       graph,
		checkStrict: true,
	}
	f.chConfigs = f.createCHConfigs()
	f.chConfig = f.chConfigs[0]
	return f
}

// createCHConfigs builds a list of distinct CH configs with different u-turn costs,
// mirroring Java CHTurnCostTest.createCHConfigs.
//   - index 0: infinite u-turn cost (default)
//   - index 1: zero u-turn cost
//   - index 2: u-turn cost of 50
//
// Additional profiles with deterministic costs are appended so callers can pick from
// `chConfigs.get(2 + rnd.nextInt(...))` style code.
func (f *preparedCHTurnCostFixture) createCHConfigs() []*CHConfig {
	tcs := f.graph.GetTurnCostStorage()
	na := f.graph.GetNodeAccess()
	configs := []*CHConfig{
		NewCHConfigEdgeBased("p0", weighting.NewSpeedWeightingWithTurnCosts(f.speedEnc, f.turnCostEnc, tcs, na, math.Inf(1))),
		NewCHConfigEdgeBased("p1", weighting.NewSpeedWeightingWithTurnCosts(f.speedEnc, f.turnCostEnc, tcs, na, 0)),
		NewCHConfigEdgeBased("p2", weighting.NewSpeedWeightingWithTurnCosts(f.speedEnc, f.turnCostEnc, tcs, na, 50)),
	}
	// add three more distinct deterministic u-turn-cost profiles
	rnd := rand.New(rand.NewSource(123))
	for len(configs) < 6 {
		uTurn := float64(10 + rnd.Intn(90))
		configs = append(configs, NewCHConfigEdgeBased(
			"p"+fmtItoa(len(configs)),
			weighting.NewSpeedWeightingWithTurnCosts(f.speedEnc, f.turnCostEnc, tcs, na, uTurn),
		))
	}
	return configs
}

// fmtItoa returns the decimal string for small non-negative integers without dragging in fmt.
func fmtItoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}

func (f *preparedCHTurnCostFixture) edge(from, to int) util.EdgeIteratorState {
	iter := f.graph.CreateEdgeExplorer(routingutil.AllEdges).SetBaseNode(from)
	for iter.Next() {
		if iter.GetAdjNode() == to {
			return iter
		}
	}
	panic("edge not found")
}

func (f *preparedCHTurnCostFixture) setTurnCost(from, via, to int, cost float64) {
	fromEdge := f.edge(from, via).GetEdge()
	toEdge := f.edge(via, to).GetEdge()
	f.graph.GetTurnCostStorage().SetDecimal(f.graph.GetNodeAccess(), f.turnCostEnc, fromEdge, via, toEdge, cost)
}

func (f *preparedCHTurnCostFixture) setTurnCostEdges(fromEdge, via, toEdge int, cost float64) {
	f.graph.GetTurnCostStorage().SetDecimal(f.graph.GetNodeAccess(), f.turnCostEnc, fromEdge, via, toEdge, cost)
}

func (f *preparedCHTurnCostFixture) setRestriction(from, via, to int) {
	f.setTurnCost(from, via, to, math.Inf(1))
}

func (f *preparedCHTurnCostFixture) setRestrictionEdges(fromEdge, via, toEdge int) {
	f.setTurnCostEdges(fromEdge, via, toEdge, math.Inf(1))
}

// prepareCH runs the real CH preparation with a fixed contraction order, populating f.chGraph.
// Mirrors Java CHTurnCostTest.prepareCH.
func (f *preparedCHTurnCostFixture) prepareCH(order ...int) {
	if !f.graph.IsFrozen() {
		f.graph.Freeze()
	}
	prepare := FromGraph(f.graph, f.chConfig).
		UseFixedNodeOrdering(NodeOrderingFromArray(order...))
	res := prepare.DoWork()
	f.chGraph = storage.NewRoutingCHGraph(f.graph, res.GetCHStorage(), f.chConfig.GetWeighting())
}

// automaticPrepareCH runs CH preparation with heuristic ordering. Mirrors Java CHTurnCostTest.automaticPrepareCH.
// Java relies on these PMap keys being honoured by SetParams; the gohopper SetParams reads the same keys
// (see PrepareContractionHierarchies.SetParams in prepare_contraction_hierarchies.go).
// Scaffolding for follow-up worktrees that port the heuristic-ordering CHTurnCostTest cases; not used here yet.
//
//nolint:unused // referenced by tests ported in later worktrees
func (f *preparedCHTurnCostFixture) automaticPrepareCH() {
	if !f.graph.IsFrozen() {
		f.graph.Freeze()
	}
	pMap := webapi.NewPMap().
		PutObject(PeriodicUpdates, 20).
		PutObject(LastLazyNodesUpdates, 100).
		PutObject(NeighborUpdates, 4).
		PutObject(LogMessages, 10)
	prepare := FromGraph(f.graph, f.chConfig).SetParams(pMap)
	res := prepare.DoWork()
	f.chGraph = storage.NewRoutingCHGraph(f.graph, res.GetCHStorage(), f.chConfig.GetWeighting())
}

// createAlgo returns the bidirectional CH algorithm for the prepared graph.
// Java defaults to DIJKSTRA_BI here; we follow the same default.
func (f *preparedCHTurnCostFixture) createAlgo() routing.EdgeToEdgeRoutingAlgorithm {
	opts := webapi.NewPMap().PutObject(chRoutingAlgorithmKey, routing.AlgoDijkstraBi)
	return NewCHRoutingAlgorithmFactory(f.chGraph).CreateAlgo(opts)
}

// findPathUsingDijkstra runs an edge-based Dijkstra on the base graph, mirroring
// Java CHTurnCostTest.findPathUsingDijkstra.
func (f *preparedCHTurnCostFixture) findPathUsingDijkstra(from, to int) *routing.Path {
	return routing.NewDijkstra(f.graph, f.chConfig.GetWeighting(), routingutil.EdgeBased).CalcPath(from, to)
}

// findPathUsingCH prepares the CH with the given contraction order and then runs the query.
func (f *preparedCHTurnCostFixture) findPathUsingCH(from, to int, contractionOrder []int) *routing.Path {
	f.prepareCH(contractionOrder...)
	return f.createAlgo().CalcPath(from, to)
}

// checkPath asserts the expected path against both standard Dijkstra and prepared-CH.
// Mirrors Java CHTurnCostTest.checkPath.
func (f *preparedCHTurnCostFixture) checkPath(t *testing.T, expectedPath []int, expectedEdgeWeight, expectedTurnCosts, from, to int, contractionOrder []int) {
	t.Helper()
	f.checkPathUsingDijkstra(t, expectedPath, expectedEdgeWeight, expectedTurnCosts, from, to)
	f.checkPathUsingCH(t, expectedPath, expectedEdgeWeight, expectedTurnCosts, from, to, contractionOrder)
}

func (f *preparedCHTurnCostFixture) checkPathUsingDijkstra(t *testing.T, expectedPath []int, expectedEdgeWeight, expectedTurnCosts, from, to int) {
	t.Helper()
	path := f.findPathUsingDijkstra(from, to)
	expectedWeight := float64(expectedEdgeWeight + expectedTurnCosts)
	expectedDistance := float64(expectedEdgeWeight * 10)
	expectedTime := int64((expectedEdgeWeight + expectedTurnCosts) * 1000)
	assert.Equal(t, expectedPath, path.CalcNodes(), "Normal Dijkstra did not find expected path.")
	assert.InDelta(t, expectedWeight, path.Weight, 1e-6, "Normal Dijkstra did not calculate expected weight.")
	assert.InDelta(t, expectedDistance, path.Distance, 1e-6, "Normal Dijkstra did not calculate expected distance.")
	assert.InDelta(t, expectedTime, path.Time, 2, "Normal Dijkstra did not calculate expected time.")
}

func (f *preparedCHTurnCostFixture) checkPathUsingCH(t *testing.T, expectedPath []int, expectedEdgeWeight, expectedTurnCosts, from, to int, contractionOrder []int) {
	t.Helper()
	path := f.findPathUsingCH(from, to, contractionOrder)
	expectedWeight := float64(expectedEdgeWeight + expectedTurnCosts)
	expectedDistance := float64(expectedEdgeWeight * 10)
	expectedTime := int64((expectedEdgeWeight + expectedTurnCosts) * 1000)
	assert.Equal(t, expectedPath, path.CalcNodes(), "Contraction Hierarchies did not find expected path. contraction order=%v", contractionOrder)
	assert.InDelta(t, expectedWeight, path.Weight, 1e-6, "Contraction Hierarchies did not calculate expected weight.")
	assert.InDelta(t, expectedDistance, path.Distance, 1e-6, "Contraction Hierarchies did not calculate expected distance.")
	assert.InDelta(t, expectedTime, path.Time, 2, "Contraction Hierarchies did not calculate expected time.")
}

// compareCHQueryWithDijkstra asserts that the prepared CH algo and a standard Dijkstra
// return paths with equal weight (and distance/time when checkStrict is true).
// Mirrors Java CHTurnCostTest.compareCHQueryWithDijkstra.
func (f *preparedCHTurnCostFixture) compareCHQueryWithDijkstra(t *testing.T, from, to int) {
	t.Helper()
	dijkstraPath := f.findPathUsingDijkstra(from, to)
	chPath := f.createAlgo().CalcPath(from, to)
	disagree := math.Abs(dijkstraPath.Weight-chPath.Weight) > 1e-2
	if f.checkStrict {
		disagree = disagree ||
			math.Abs(dijkstraPath.Distance-chPath.Distance) > 1e-2 ||
			math.Abs(float64(dijkstraPath.Time-chPath.Time)) > 1
	}
	if disagree {
		t.Fatalf("Dijkstra and CH did not find equal shortest paths for route from %d to %d\n"+
			" dijkstra: weight: %v, distance: %v, time: %v, nodes: %v\n"+
			"       ch: weight: %v, distance: %v, time: %v, nodes: %v",
			from, to,
			dijkstraPath.Weight, dijkstraPath.Distance, dijkstraPath.Time, dijkstraPath.CalcNodes(),
			chPath.Weight, chPath.Distance, chPath.Time, chPath.CalcNodes())
	}
}

// compareCHWithDijkstra prepares the CH with the given contraction order, then runs
// `numQueries` random routing queries comparing CH and Dijkstra. Mirrors Java's helper of the
// same name, but uses a fixed seed for reproducibility (Go ports avoid System.nanoTime).
func (f *preparedCHTurnCostFixture) compareCHWithDijkstra(t *testing.T, numQueries int, contractionOrder []int, seed int64) {
	t.Helper()
	f.prepareCH(contractionOrder...)
	rnd := rand.New(rand.NewSource(seed))
	nodes := f.graph.GetNodes()
	for i := 0; i < numQueries; i++ {
		f.compareCHQueryWithDijkstra(t, rnd.Intn(nodes), rnd.Intn(nodes))
	}
}

// setRandomCost mirrors Java CHTurnCostTest.setRandomCost. Java draws a value in [0, maxCost/2)
// via rnd.nextDouble() * maxCost/2.
func (f *preparedCHTurnCostFixture) setRandomCost(from, via, to int, rnd *rand.Rand) {
	cost := int(rnd.Float64() * float64(f.maxCost) / 2)
	f.setTurnCost(from, via, to, float64(cost))
}

// setRandomCostOrRestriction mirrors Java CHTurnCostTest.setRandomCostOrRestriction.
// With 70% probability it sets a hard restriction, otherwise a random cost.
func (f *preparedCHTurnCostFixture) setRandomCostOrRestriction(from, via, to int, rnd *rand.Rand) {
	if rnd.Float64() < 0.7 {
		f.setRestriction(from, via, to)
	} else {
		f.setRandomCost(from, via, to, rnd)
	}
}

// --- Phase 1: deterministic prepared-CH cases ported from CHTurnCostTest ---

// TestPreparedCH_MultipleInOutEdges_TurnReplacementDifference ports Java CHTurnCostTest L160.
// Java uses @RepeatedTest(100) with System.nanoTime() seed; we use a fixed seed so failures
// are reproducible. The test exercises a graph with multiple in/out edges around node 6 where
// shortcut creation depends on the random turn restrictions placed around nodes 5 and 7.
func TestPreparedCH_MultipleInOutEdges_TurnReplacementDifference(t *testing.T) {
	f := newPreparedCHTurnCostFixture()
	//   0   3 - 4   8
	//    \ /     \ /
	// 1 - 5 - 6 - 7 - 9
	//    /         \
	//   2           10
	f.graph.Edge(0, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(5, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(4, 7).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(5, 6).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(6, 7).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(7, 8).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(7, 9).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(7, 10).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)

	rnd := rand.New(rand.NewSource(1))
	f.setRandomCost(2, 5, 3, rnd)
	f.setRandomCost(2, 5, 6, rnd)
	f.setRandomCost(4, 7, 10, rnd)
	f.setRandomCost(6, 7, 10, rnd)
	f.setRandomCostOrRestriction(0, 5, 3, rnd)
	f.setRandomCostOrRestriction(1, 5, 3, rnd)
	f.setRandomCostOrRestriction(0, 5, 6, rnd)
	f.setRandomCostOrRestriction(1, 5, 6, rnd)
	f.setRandomCostOrRestriction(4, 7, 8, rnd)
	f.setRandomCostOrRestriction(4, 7, 9, rnd)
	f.setRandomCostOrRestriction(6, 7, 8, rnd)
	f.setRandomCostOrRestriction(6, 7, 9, rnd)

	f.prepareCH(6, 0, 1, 2, 8, 9, 10, 5, 3, 4, 7)
	f.checkStrict = false
	f.compareCHQueryWithDijkstra(t, 2, 10)
	f.compareCHQueryWithDijkstra(t, 1, 10)
	f.compareCHQueryWithDijkstra(t, 2, 9)
	f.compareCHQueryWithDijkstra(t, 1, 9)
}

// TestPreparedCH_MultipleInOutEdges_TurnReplacementDifference_Bug1 ports Java CHTurnCostTest L210.
// Hand-tuned regression seed for the @RepeatedTest above.
func TestPreparedCH_MultipleInOutEdges_TurnReplacementDifference_Bug1(t *testing.T) {
	f := newPreparedCHTurnCostFixture()
	//       3 - 4
	//      /     \
	// 1 - 5 - 6 - 7 - 9
	//    /         \
	//   2           10
	f.graph.Edge(1, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(5, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(4, 7).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(5, 6).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(6, 7).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(7, 9).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(7, 10).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)

	f.setTurnCost(2, 5, 6, 4)
	f.setRestriction(1, 5, 6)
	f.setRestriction(4, 7, 9)

	f.prepareCH(6, 0, 1, 2, 8, 9, 10, 5, 3, 4, 7)
	f.compareCHQueryWithDijkstra(t, 2, 9)
}

// TestPreparedCH_DuplicateEdge ports Java CHTurnCostTest L235.
// 0 -> 1 -> 2 -> 3 -> 4 with a duplicate 2->3 edge.
func TestPreparedCH_DuplicateEdge(t *testing.T) {
	f := newPreparedCHTurnCostFixture()
	f.graph.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.compareCHWithDijkstra(t, 100, []int{2, 3, 0, 4, 1}, 1)
}

// TestPreparedCH_Chain ports Java CHTurnCostTest L247.
// Chain 0-1-...-8 with turn costs at nodes 2/4/6 and a non-trivial contraction order
// that forces fwd/bwd searches to meet at node 4.
func TestPreparedCH_Chain(t *testing.T) {
	f := newPreparedCHTurnCostFixture()
	// 0   2   4   6   8
	//  \ / \ / \ / \ /
	//   1   3   5   7
	f.graph.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(4, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(5, 6).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(6, 7).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(7, 8).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Freeze()
	f.setTurnCost(1, 2, 3, 4)
	f.setTurnCost(3, 4, 5, 2)
	f.setTurnCost(5, 6, 7, 3)

	f.checkPathUsingCH(t, []int{0, 1, 2, 3, 4, 5, 6, 7, 8}, 8, 9, 0, 8, []int{1, 3, 5, 7, 0, 8, 2, 6, 4})
}

// TestPreparedCH_BidirChain ports Java CHTurnCostTest L271.
// 7-node chain with different forward and backward turn costs at every node.
func TestPreparedCH_BidirChain(t *testing.T) {
	f := newPreparedCHTurnCostFixture()
	//   5 3 2 1 4    turn costs ->
	// 0-1-2-3-4-5-6
	//   0 1 4 2 3    turn costs <-
	edge0 := f.graph.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	edge1 := f.graph.Edge(1, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	edge2 := f.graph.Edge(2, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	edge3 := f.graph.Edge(3, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	edge4 := f.graph.Edge(4, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	edge5 := f.graph.Edge(5, 6).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Freeze()

	// turn costs ->
	f.setTurnCostEdges(edge0.GetEdge(), 1, edge1.GetEdge(), 5)
	f.setTurnCostEdges(edge1.GetEdge(), 2, edge2.GetEdge(), 3)
	f.setTurnCostEdges(edge2.GetEdge(), 3, edge3.GetEdge(), 2)
	f.setTurnCostEdges(edge3.GetEdge(), 4, edge4.GetEdge(), 1)
	f.setTurnCostEdges(edge4.GetEdge(), 5, edge5.GetEdge(), 4)
	// turn costs <-
	f.setTurnCostEdges(edge5.GetEdge(), 5, edge4.GetEdge(), 3)
	f.setTurnCostEdges(edge4.GetEdge(), 4, edge3.GetEdge(), 2)
	f.setTurnCostEdges(edge3.GetEdge(), 3, edge2.GetEdge(), 4)
	f.setTurnCostEdges(edge2.GetEdge(), 2, edge1.GetEdge(), 1)
	f.setTurnCostEdges(edge1.GetEdge(), 1, edge0.GetEdge(), 0)

	f.prepareCH(1, 3, 5, 2, 4, 0, 6)

	pathFwd := f.createAlgo().CalcPath(0, 6)
	assert.Equal(t, []int{0, 1, 2, 3, 4, 5, 6}, pathFwd.CalcNodes())
	assert.InDelta(t, float64(6+15), pathFwd.Weight, 1e-6)

	pathBwd := f.createAlgo().CalcPath(6, 0)
	assert.Equal(t, []int{6, 5, 4, 3, 2, 1, 0}, pathBwd.CalcNodes())
	assert.InDelta(t, float64(6+10), pathBwd.Weight, 1e-6)
}

// TestPreparedCH_PTurn_UTurnAtContractedNode ports Java CHTurnCostTest L534.
// Contracting node 4 forces a loop shortcut at node 6.
func TestPreparedCH_PTurn_UTurnAtContractedNode(t *testing.T) {
	f := newPreparedCHTurnCostFixture()
	//           2- 3
	//           |  |
	//           4- 0
	//           |
	//     5 ->  6 -> 1
	f.graph.Edge(5, 6).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(6, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(6, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(4, 0).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(0, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Freeze()
	f.setRestriction(5, 6, 1)

	expectedPath := []int{5, 6, 4, 0, 3, 2, 4, 6, 1}
	f.checkPath(t, expectedPath, 8, 0, 5, 1, []int{0, 1, 2, 3, 4, 5, 6})
}

// TestPreparedCH_PTurn_UTurnAtContractedNode_TwoShortcutsInAndOut ports Java CHTurnCostTest L557.
func TestPreparedCH_PTurn_UTurnAtContractedNode_TwoShortcutsInAndOut(t *testing.T) {
	f := newPreparedCHTurnCostFixture()
	//           2- 3
	//           |  |
	//           4- 0
	//           |
	//           1
	//           |
	//     5 ->  6 -> 7
	f.graph.Edge(5, 6).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(6, 7).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(6, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(1, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(4, 0).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(0, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Freeze()
	f.setRestriction(5, 6, 7)

	expectedPath := []int{5, 6, 1, 4, 0, 3, 2, 4, 1, 6, 7}
	f.checkPath(t, expectedPath, 10, 0, 5, 7, []int{0, 1, 2, 3, 4, 5, 6, 7})
}

// TestPreparedCH_Bug ports Java CHTurnCostTest L664. Regression for an earlier CH bug.
func TestPreparedCH_Bug(t *testing.T) {
	f := newPreparedCHTurnCostFixture()
	f.graph.Edge(1, 2).SetDistance(180.364).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 4).SetDistance(290.814).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(0, 2).SetDistance(140.554).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(1, 4).SetDistance(290.819).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(1, 3).SetDistance(290.271).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.setRestriction(3, 1, 2)
	f.graph.Freeze()

	f.compareCHWithDijkstra(t, 100, []int{1, 0, 3, 2, 4}, 1)
}

// TestPreparedCH_Bug2 ports Java CHTurnCostTest L677. Regression for another CH bug with duplicate edges.
func TestPreparedCH_Bug2(t *testing.T) {
	f := newPreparedCHTurnCostFixture()
	// 1 = 0 - 3 - 2 - 4
	f.graph.Edge(0, 3).SetDistance(240.001).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(0, 1).SetDistance(60.087).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(0, 1).SetDistance(60.067).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(2, 3).SetDistance(460.631).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(2, 4).SetDistance(460.184).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Freeze()

	f.compareCHWithDijkstra(t, 1000, []int{1, 0, 3, 2, 4}, 1)
}

// TestPreparedCH_Loop ports Java CHTurnCostTest L690.
func TestPreparedCH_Loop(t *testing.T) {
	f := newPreparedCHTurnCostFixture()
	//             3
	//            / \
	//           1   2
	//            \ /
	// 0 - 7 - 8 - 4 - 6 - 5
	f.graph.Edge(0, 7).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(7, 8).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(8, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(4, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(4, 6).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(6, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.setRestriction(8, 4, 6)
	f.graph.Freeze()

	f.prepareCH(0, 1, 2, 3, 4, 5, 6, 7, 8)
	f.compareCHQueryWithDijkstra(t, 0, 5)
}

// TestPreparedCH_FiniteUTurnCost ports Java CHTurnCostTest L713.
// Turning to node 1 at node 3 when coming from 0 is forbidden, but the loop 3-4-2-3 is expensive,
// so the best solution is to take a u-turn at node 4 (which has finite u-turn cost in chConfigs[2]).
func TestPreparedCH_FiniteUTurnCost(t *testing.T) {
	f := newPreparedCHTurnCostFixture()
	//   1
	//   |
	// 0-3-4
	//   |/
	//   2
	f.graph.Edge(0, 3).SetDistance(1000).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 4).SetDistance(1000).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(4, 2).SetDistance(5000).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 3).SetDistance(2000).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 1).SetDistance(1000).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.setRestriction(0, 3, 1)
	f.graph.Freeze()
	f.chConfig = f.chConfigs[2]
	f.prepareCH(4, 0, 2, 3, 1)
	path := f.createAlgo().CalcPath(0, 1)
	assert.Equal(t, []int{0, 3, 4, 3, 1}, path.CalcNodes())
	f.compareCHQueryWithDijkstra(t, 0, 1)
}

// TestPreparedCH_CalcTurnCostTime ports Java CHTurnCostTest L736.
// When the path is unpacked it is important that turn costs are included at node 1 even
// though the unpacked original edge 1->0 might be in the reverted state.
func TestPreparedCH_CalcTurnCostTime(t *testing.T) {
	f := newPreparedCHTurnCostFixture()
	// 2-1--3
	//   |  |
	//   0->4
	edge0 := f.graph.Edge(1, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(0, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(4, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	edge3 := f.graph.Edge(1, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	edge4 := f.graph.Edge(1, 0).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.setTurnCostEdges(edge0.GetEdge(), 1, edge4.GetEdge(), 8)
	f.setRestrictionEdges(edge0.GetEdge(), 1, edge3.GetEdge())
	f.graph.Freeze()
	f.checkPath(t, []int{2, 1, 0, 4}, 3, 8, 2, 4, []int{2, 0, 1, 3, 4})
}
