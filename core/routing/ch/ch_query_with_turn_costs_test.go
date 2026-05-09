package ch

import (
	"math"
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
	f.graph.GetTurnCostStorage().SetDecimal(f.graph.GetNodeAccess(), f.turnCostEnc, fromEdge, via, toEdge, cost)
}

func (f *turnCostQueryFixture) setRestriction(from, via, to int) {
	f.setTurnCost(from, via, to, math.Inf(1))
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
