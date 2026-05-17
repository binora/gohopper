package ch

import (
	"math"
	"math/rand"
	"strconv"
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

// -----------------------------------------------------------------------------
// Dijkstra-comparison stress tests (gohopper-9x7.6).
//
// Ports the six @RepeatedTest(10) cases from
// core/src/test/java/com/graphhopper/routing/ch/CHTurnCostTest.java:
//
//	L581  testFindPath_highlyConnectedGraph_compareWithDijkstra
//	L1059 testFindPath_random_compareWithDijkstra
//	L1066 testFindPath_random_compareWithDijkstra_finiteUTurnCost
//	L1074 testFindPath_random_compareWithDijkstra_zeroUTurnCost
//	L1096 testFindPath_heuristic_compareWithDijkstra
//	L1103 testFindPath_heuristic_compareWithDijkstra_finiteUTurnCost
//
// Each @RepeatedTest(10) becomes 10 seeded t.Run subtests with seeds
// int64(0)..int64(9).
// -----------------------------------------------------------------------------

// chStressFixture mirrors the Java CHTurnCostTest @BeforeEach state. The
// chConfig field is mutable so finiteUTurnCost / zeroUTurnCost variants
// can rebind it between graph build and CH preparation, matching Java
// lines 1069, 1077, and 1106.
type chStressFixture struct {
	maxCost     int
	speedEnc    ev.DecimalEncodedValue
	turnCostEnc ev.DecimalEncodedValue
	graph       *storage.BaseGraph
	chConfigs   []*CHConfig
	chConfig    *CHConfig
	checkStrict bool
}

func newCHStressFixture(seed int64) *chStressFixture {
	maxCost := 10
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	turnCostEnc := ev.TurnCostCreate("car", maxCost)
	em := routingutil.Start().Add(speedEnc).AddTurnCostEncodedValue(turnCostEnc).Build()
	g := storage.NewBaseGraphBuilder(em.BytesForFlags).SetWithTurnCosts(true).CreateGraph()

	configs := chStressBuildConfigs(speedEnc, turnCostEnc, g, seed)
	return &chStressFixture{
		maxCost:     maxCost,
		speedEnc:    speedEnc,
		turnCostEnc: turnCostEnc,
		graph:       g,
		chConfigs:   configs,
		chConfig:    configs[0],
		checkStrict: true,
	}
}

// chStressBuildConfigs mirrors Java createCHConfigs: 3 fixed profiles
// (infinite, 0, 50 u-turn cost) plus random profiles in [10, 100) until
// the set holds 6 distinct configs. The seed makes the random profiles
// reproducible within a subtest.
func chStressBuildConfigs(speedEnc, turnCostEnc ev.DecimalEncodedValue, g *storage.BaseGraph, seed int64) []*CHConfig {
	makeConfig := func(name string, uTurnCosts float64) *CHConfig {
		w := weighting.NewSpeedWeightingWithTurnCosts(speedEnc, turnCostEnc, g.GetTurnCostStorage(), g.GetNodeAccess(), uTurnCosts)
		return NewCHConfigEdgeBased(name, w)
	}
	configs := []*CHConfig{
		makeConfig("p0", math.Inf(1)),
		makeConfig("p1", 0),
		makeConfig("p2", 50),
	}
	seen := map[float64]bool{math.Inf(1): true, 0: true, 50: true}
	rnd := rand.New(rand.NewSource(seed))
	for len(configs) < 6 {
		u := float64(10 + rnd.Intn(90))
		if seen[u] {
			continue
		}
		seen[u] = true
		configs = append(configs, makeConfig("p"+strconv.Itoa(len(configs)), u))
	}
	return configs
}

// shuffleIota returns a randomized permutation of [0, n). Mirrors Java
// ArrayUtil.shuffle(ArrayUtil.iota(n), rnd).
func shuffleIota(n int, rnd *rand.Rand) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	rnd.Shuffle(n, func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

// findPathUsingDijkstra mirrors Java findPathUsingDijkstra: edge-based
// Dijkstra over the BaseGraph using the current chConfig weighting.
func (f *chStressFixture) findPathUsingDijkstra(from, to int) *routing.Path {
	d := routing.NewDijkstra(f.graph, f.chConfig.GetWeighting(), routingutil.EdgeBased)
	return d.CalcPath(from, to)
}

// prepareCHFixed mirrors Java prepareCH(contractionOrder): freezes the
// graph if needed and runs CH preparation with the given fixed order.
func (f *chStressFixture) prepareCHFixed(order []int) storage.RoutingCHGraph {
	if !f.graph.IsFrozen() {
		f.graph.Freeze()
	}
	prepare := FromGraph(f.graph, f.chConfig).UseFixedNodeOrdering(NodeOrderingFromArray(order...))
	res := prepare.DoWork()
	return storage.NewRoutingCHGraph(f.graph, res.GetCHStorage(), res.GetCHConfig().GetWeighting())
}

// prepareCHAutomatic mirrors Java automaticPrepareCH: heuristic node
// priority with the same PMap tuning constants used in Java.
func (f *chStressFixture) prepareCHAutomatic() storage.RoutingCHGraph {
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
	return storage.NewRoutingCHGraph(f.graph, res.GetCHStorage(), res.GetCHConfig().GetWeighting())
}

// chStressPath runs a Dijkstra-bi CH query over the prepared graph.
func chStressPath(chGraph storage.RoutingCHGraph, from, to int) *routing.Path {
	opts := webapi.NewPMap().PutObject(chRoutingAlgorithmKey, routing.AlgoDijkstraBi)
	return NewCHRoutingAlgorithmFactory(chGraph).CreateAlgo(opts).CalcPath(from, to)
}

// compareCHWithDijkstra mirrors Java compareCHWithDijkstra. It prepares
// CH with a fixed contraction order, then runs numQueries random
// (from, to) pairs and asserts CH and Dijkstra agree on weight (and
// distance/time when checkStrict).
func (f *chStressFixture) compareCHWithDijkstra(t *testing.T, numQueries int, order []int, querySeed int64) {
	t.Helper()
	chGraph := f.prepareCHFixed(order)
	rnd := rand.New(rand.NewSource(querySeed))
	n := f.graph.GetNodes()
	for i := 0; i < numQueries; i++ {
		f.compareCHQueryWithDijkstra(t, chGraph, rnd.Intn(n), rnd.Intn(n))
	}
}

// automaticCompareCHWithDijkstra mirrors Java automaticCompareCHWithDijkstra.
func (f *chStressFixture) automaticCompareCHWithDijkstra(t *testing.T, numQueries int, querySeed int64) {
	t.Helper()
	chGraph := f.prepareCHAutomatic()
	rnd := rand.New(rand.NewSource(querySeed))
	n := f.graph.GetNodes()
	for i := 0; i < numQueries; i++ {
		f.compareCHQueryWithDijkstra(t, chGraph, rnd.Intn(n), rnd.Intn(n))
	}
}

// compareCHQueryWithDijkstra mirrors Java compareCHQueryWithDijkstra. It
// runs both algorithms for the given pair and asserts they agree. When
// checkStrict is false only weights are compared (matching Java).
func (f *chStressFixture) compareCHQueryWithDijkstra(t *testing.T, chGraph storage.RoutingCHGraph, from, to int) {
	t.Helper()
	dPath := f.findPathUsingDijkstra(from, to)
	chPath := chStressPath(chGraph, from, to)
	disagree := math.Abs(dPath.Weight-chPath.Weight) > 1e-2
	if f.checkStrict {
		disagree = disagree ||
			math.Abs(dPath.Distance-chPath.Distance) > 1e-2 ||
			math.Abs(float64(dPath.Time-chPath.Time)) > 1
	}
	if disagree {
		assert.Failf(t, "Dijkstra and CH disagree",
			"from=%d to=%d dijkstra(w=%g,d=%g,t=%d nodes=%v) ch(w=%g,d=%g,t=%d nodes=%v)",
			from, to, dPath.Weight, dPath.Distance, dPath.Time, dPath.CalcNodes(),
			chPath.Weight, chPath.Distance, chPath.Time, chPath.CalcNodes())
	}
}

// chStressNextCost mirrors Java nextCost.
func chStressNextCost(rnd *rand.Rand, maxCost int) int { return rnd.Intn(3 * maxCost) }

// chStressNextDist mirrors Java nextDist.
func chStressNextDist(rnd *rand.Rand, maxDist int) float64 { return rnd.Float64() * float64(maxDist) }

// chStressSetCostOrRestriction mirrors Java setCostOrRestriction.
func (f *chStressFixture) chStressSetCostOrRestriction(inEdge, viaNode, outEdge, cost int) {
	if cost >= f.maxCost {
		f.graph.GetTurnCostStorage().SetDecimal(f.graph.GetNodeAccess(), f.turnCostEnc, inEdge, viaNode, outEdge, math.Inf(1))
	} else {
		f.graph.GetTurnCostStorage().SetDecimal(f.graph.GetNodeAccess(), f.turnCostEnc, inEdge, viaNode, outEdge, float64(cost))
	}
}

// stressRandomQueries / stressHighlyConnectedQueries mirror Java's
// 100 / 1000 numQueries values. If the 30s runtime budget is exceeded,
// reduce these here; the plan forbids dropping subtests.
const (
	stressRandomQueries          = 100
	stressHighlyConnectedQueries = 1000
)

func TestCHTurnCost_FindPath_HighlyConnectedGraph_CompareWithDijkstra(t *testing.T) {
	for i := 0; i < 10; i++ {
		seed := int64(i)
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			f := newCHStressFixture(seed)
			rnd := rand.New(rand.NewSource(seed))

			const (
				size    = 4
				maxDist = 40
			)
			// horizontal edges
			for i := 0; i < size; i++ {
				for j := 0; j < size-1; j++ {
					from := i*size + j
					to := from + 1
					f.graph.Edge(from, to).SetDistance(chStressNextDist(rnd, maxDist)).SetDecimalBothDir(f.speedEnc, 10, 10)
				}
			}
			// vertical edges
			for i := 0; i < size-1; i++ {
				for j := 0; j < size; j++ {
					from := i*size + j
					to := from + size
					f.graph.Edge(from, to).SetDistance(chStressNextDist(rnd, maxDist)).SetDecimalBothDir(f.speedEnc, 10, 10)
				}
			}
			// diagonal edges
			for i := 0; i < size-1; i++ {
				for j := 0; j < size; j++ {
					from := i*size + j
					if j < size-1 {
						f.graph.Edge(from, from+size+1).SetDistance(chStressNextDist(rnd, maxDist)).SetDecimalBothDir(f.speedEnc, 10, 10)
					}
					if j > 0 {
						f.graph.Edge(from, from+size-1).SetDistance(chStressNextDist(rnd, maxDist)).SetDecimalBothDir(f.speedEnc, 10, 10)
					}
				}
			}
			f.graph.Freeze()

			// turn costs / restrictions on every (in, out) pair, skipping u-turns
			inExplorer := f.graph.CreateEdgeExplorer(routingutil.AllEdges)
			outExplorer := f.graph.CreateEdgeExplorer(routingutil.AllEdges)
			for node := 0; node < size*size; node++ {
				inIter := inExplorer.SetBaseNode(node)
				for inIter.Next() {
					outIter := outExplorer.SetBaseNode(node)
					for outIter.Next() {
						if inIter.GetEdge() == outIter.GetEdge() {
							continue
						}
						f.chStressSetCostOrRestriction(inIter.GetEdge(), node, outIter.GetEdge(), chStressNextCost(rnd, f.maxCost))
					}
				}
			}

			order := shuffleIota(f.graph.GetNodes(), rnd)
			f.checkStrict = false
			f.compareCHWithDijkstra(t, stressHighlyConnectedQueries, order, seed)
		})
	}
}

func TestCHTurnCost_FindPath_Random_CompareWithDijkstra(t *testing.T) {
	for i := 0; i < 10; i++ {
		seed := int64(i)
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			chStressRunRandom(t, seed, 0 /* default config */)
		})
	}
}

func TestCHTurnCost_FindPath_Random_CompareWithDijkstra_FiniteUTurnCost(t *testing.T) {
	for i := 0; i < 10; i++ {
		seed := int64(i)
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			// Java: chConfig = chConfigs.get(2 + rnd.nextInt(chConfigs.size() - 2))
			cfgPicker := rand.New(rand.NewSource(seed))
			chStressRunRandom(t, seed, 2+cfgPicker.Intn(4))
		})
	}
}

func TestCHTurnCost_FindPath_Random_CompareWithDijkstra_ZeroUTurnCost(t *testing.T) {
	for i := 0; i < 10; i++ {
		seed := int64(i)
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			// Java: chConfig = chConfigs.get(1)
			chStressRunRandom(t, seed, 1)
		})
	}
}

// chStressRunRandom mirrors Java compareWithDijkstraOnRandomGraph. The
// chConfigIdx argument lets the three random subtests switch profiles
// between graph build and CH prepare (Java lines 1069 / 1077).
func chStressRunRandom(t *testing.T, seed int64, chConfigIdx int) {
	t.Helper()
	f := newCHStressFixture(seed)
	graphRnd := rand.New(rand.NewSource(seed))
	util.RandomGraph(f.graph.GetNodeAccess(), f.graph.Edge, graphRnd, 20, 3.0, true, f.speedEnc, nil, 0.9, 0.8)
	tcs := f.graph.GetTurnCostStorage()
	turnRnd := rand.New(rand.NewSource(seed))
	util.AddRandomTurnCosts(
		f.graph.GetNodes(), turnRnd,
		f.graph.CreateEdgeExplorer(routingutil.AllEdges),
		f.graph.CreateEdgeExplorer(routingutil.AllEdges),
		f.turnCostEnc, f.maxCost,
		func(enc ev.DecimalEncodedValue, fromEdge, viaNode, toEdge int, cost float64) {
			tcs.SetDecimal(f.graph.GetNodeAccess(), enc, fromEdge, viaNode, toEdge, cost)
		})
	f.graph.Freeze()
	f.chConfig = f.chConfigs[chConfigIdx]
	f.checkStrict = false
	order := shuffleIota(f.graph.GetNodes(), graphRnd)
	f.compareCHWithDijkstra(t, stressRandomQueries, order, seed)
}

func TestCHTurnCost_FindPath_Heuristic_CompareWithDijkstra(t *testing.T) {
	for i := 0; i < 10; i++ {
		seed := int64(i)
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			chStressRunHeuristic(t, seed, 0 /* default config */)
		})
	}
}

func TestCHTurnCost_FindPath_Heuristic_CompareWithDijkstra_FiniteUTurnCost(t *testing.T) {
	for i := 0; i < 10; i++ {
		seed := int64(i)
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			cfgPicker := rand.New(rand.NewSource(seed))
			chStressRunHeuristic(t, seed, 2+cfgPicker.Intn(4))
		})
	}
}

// chStressRunHeuristic mirrors Java compareWithDijkstraOnRandomGraph_heuristic.
func chStressRunHeuristic(t *testing.T, seed int64, chConfigIdx int) {
	t.Helper()
	f := newCHStressFixture(seed)
	rnd := rand.New(rand.NewSource(seed))
	util.RandomGraph(f.graph.GetNodeAccess(), f.graph.Edge, rnd, 20, 3.0, true, f.speedEnc, nil, 0.9, 0.8)
	tcs := f.graph.GetTurnCostStorage()
	turnRnd := rand.New(rand.NewSource(seed))
	util.AddRandomTurnCosts(
		f.graph.GetNodes(), turnRnd,
		f.graph.CreateEdgeExplorer(routingutil.AllEdges),
		f.graph.CreateEdgeExplorer(routingutil.AllEdges),
		f.turnCostEnc, f.maxCost,
		func(enc ev.DecimalEncodedValue, fromEdge, viaNode, toEdge int, cost float64) {
			tcs.SetDecimal(f.graph.GetNodeAccess(), enc, fromEdge, viaNode, toEdge, cost)
		})
	f.graph.Freeze()
	f.chConfig = f.chConfigs[chConfigIdx]
	f.checkStrict = false
	f.automaticCompareCHWithDijkstra(t, stressRandomQueries, seed)
}
