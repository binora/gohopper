package storage_test

import (
	"math"
	"testing"

	"gohopper/core/routing"
	"gohopper/core/routing/ch"
	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"

	"github.com/stretchr/testify/assert"
)

const (
	prevEdge = 12
	nextEdge = 13
)

type unpackFixture struct {
	edgeBased   bool
	speedEnc    ev.DecimalEncodedValue
	turnCostEnc ev.DecimalEncodedValue
	graph       *storage.BaseGraph
	chBuilder   *storage.CHStorageBuilder
	chGraph     storage.RoutingCHGraph
}

func newUnpackFixture(edgeBased bool) *unpackFixture {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	turnCostEnc := ev.TurnCostCreate("car", 10)
	em := routingutil.Start().Add(speedEnc).AddTurnCostEncodedValue(turnCostEnc).Build()
	g := storage.NewBaseGraphBuilder(em.BytesForFlags).SetWithTurnCosts(true).CreateGraph()
	return &unpackFixture{
		edgeBased:   edgeBased,
		speedEnc:    speedEnc,
		turnCostEnc: turnCostEnc,
		graph:       g,
	}
}

func (f *unpackFixture) freeze() {
	f.graph.Freeze()
	var w storage.CHWeighting
	if f.edgeBased {
		w = weighting.NewSpeedWeightingWithTurnCosts(f.speedEnc, f.turnCostEnc, f.graph.GetTurnCostStorage(), f.graph.GetNodeAccess(), math.Inf(1))
	} else {
		w = weighting.NewSpeedWeighting(f.speedEnc)
	}
	store := storage.CHStorageFromGraph(f.graph, "profile", f.edgeBased)
	f.chBuilder = storage.NewCHStorageBuilder(store)
	f.chGraph = storage.NewRoutingCHGraph(f.graph, store, w)
}

func (f *unpackFixture) setCHLevels(order ...int) {
	for i, node := range order {
		f.chBuilder.SetLevel(node, i)
	}
}

func (f *unpackFixture) shortcut(baseNode, adjNode, skip1, skip2, origKeyFirst, origKeyLast int, reverse bool) {
	weight := 1.0
	flags := ch.ScBwdDir
	if reverse {
		flags = ch.ScFwdDir
	}
	if f.edgeBased {
		f.chBuilder.AddShortcutEdgeBased(baseNode, adjNode, flags, weight, skip1, skip2, origKeyFirst, origKeyLast)
	} else {
		f.chBuilder.AddShortcutNodeBased(baseNode, adjNode, flags, weight, skip1, skip2)
	}
}

func (f *unpackFixture) setTurnCost(fromEdge, viaNode, toEdge int, cost float64) {
	f.graph.GetTurnCostStorage().SetDecimal(f.graph.GetNodeAccess(), f.turnCostEnc, fromEdge, viaNode, toEdge, cost)
}

func (f *unpackFixture) visitFwd(edge, adj int, reverseOrder bool, visitor ch.Visitor) {
	ch.NewShortcutUnpacker(f.chGraph, visitor, f.edgeBased).VisitOriginalEdgesFwd(edge, adj, reverseOrder, prevEdge)
}

func (f *unpackFixture) visitBwd(edge, adjNode int, reverseOrder bool, visitor ch.Visitor) {
	ch.NewShortcutUnpacker(f.chGraph, visitor, f.edgeBased).VisitOriginalEdgesBwd(edge, adjNode, reverseOrder, nextEdge)
}

// testVisitor collects edge data for each visited original edge.
type testVisitor struct {
	w                  weighting.Weighting
	edgeIds            []int
	baseNodes          []int
	adjNodes           []int
	prevOrNextEdgeIds  []int
	weights            []float64
	distances          []float64
	times              []float64
}

func newTestVisitor(chGraph storage.RoutingCHGraph) *testVisitor {
	return &testVisitor{w: chGraph.GetWeighting().(weighting.Weighting)}
}

func (v *testVisitor) Visit(edge util.EdgeIteratorState, reverse bool, prevOrNextEdgeId int) {
	v.edgeIds = append(v.edgeIds, edge.GetEdge())
	v.baseNodes = append(v.baseNodes, edge.GetBaseNode())
	v.adjNodes = append(v.adjNodes, edge.GetAdjNode())
	v.weights = append(v.weights, routing.CalcWeightWithTurnWeight(v.w, edge, reverse, prevOrNextEdgeId))
	v.distances = append(v.distances, edge.GetDistance())
	v.times = append(v.times, float64(routing.CalcMillisWithTurnMillis(v.w, edge, reverse, prevOrNextEdgeId)))
	v.prevOrNextEdgeIds = append(v.prevOrNextEdgeIds, prevOrNextEdgeId)
}

// turnWeightingVisitor accumulates total weight and time.
type turnWeightingVisitor struct {
	w      weighting.Weighting
	weight float64
	time   int64
}

func newTurnWeightingVisitor(chGraph storage.RoutingCHGraph) *turnWeightingVisitor {
	return &turnWeightingVisitor{w: chGraph.GetWeighting().(weighting.Weighting)}
}

func (v *turnWeightingVisitor) Visit(edge util.EdgeIteratorState, reverse bool, prevOrNextEdgeId int) {
	v.time += routing.CalcMillisWithTurnMillis(v.w, edge, reverse, prevOrNextEdgeId)
	v.weight += routing.CalcWeightWithTurnWeight(v.w, edge, reverse, prevOrNextEdgeId)
}

func TestShortcutUnpacker_Unpacking(t *testing.T) {
	for _, edgeBased := range []bool{false, true} {
		name := "node-based"
		if edgeBased {
			name = "edge-based"
		}
		t.Run(name, func(t *testing.T) {
			f := newUnpackFixture(edgeBased)
			// 0-1-2-3-4-5-6
			f.graph.Edge(0, 1).SetDistance(1).SetDecimalBothDir(f.speedEnc, 20, 10)
			f.graph.Edge(1, 2).SetDistance(1).SetDecimalBothDir(f.speedEnc, 20, 10)
			f.graph.Edge(2, 3).SetDistance(1).SetDecimalBothDir(f.speedEnc, 20, 10)
			f.graph.Edge(3, 4).SetDistance(1).SetDecimalBothDir(f.speedEnc, 20, 10)
			f.graph.Edge(4, 5).SetDistance(1).SetDecimalBothDir(f.speedEnc, 20, 10)
			f.graph.Edge(5, 6).SetDistance(1).SetDecimalBothDir(f.speedEnc, 20, 10)
			f.freeze()

			f.setCHLevels(1, 3, 5, 4, 2, 0, 6)
			f.shortcut(4, 2, 2, 3, 4, 6, true)
			f.shortcut(4, 6, 4, 5, 8, 10, false)
			f.shortcut(2, 0, 0, 1, 0, 2, true)
			f.shortcut(2, 6, 6, 7, 4, 10, false)
			f.shortcut(0, 6, 8, 9, 0, 10, false)

			// forward, normal order
			v := newTestVisitor(f.chGraph)
			f.visitFwd(10, 6, false, v)
			assert.Equal(t, []int{0, 1, 2, 3, 4, 5}, v.edgeIds)
			assert.Equal(t, []int{0, 1, 2, 3, 4, 5}, v.baseNodes)
			assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, v.adjNodes)
			assert.InDeltaSlice(t, []float64{0.05, 0.05, 0.05, 0.05, 0.05, 0.05}, v.weights, 1e-6)
			assert.InDeltaSlice(t, []float64{1, 1, 1, 1, 1, 1}, v.distances, 1e-6)
			assert.InDeltaSlice(t, []float64{50, 50, 50, 50, 50, 50}, v.times, 1e-6)
			if edgeBased {
				assert.Equal(t, []int{prevEdge, 0, 1, 2, 3, 4}, v.prevOrNextEdgeIds)
			}

			// forward, reverse order
			v = newTestVisitor(f.chGraph)
			f.visitFwd(10, 6, true, v)
			assert.Equal(t, []int{5, 4, 3, 2, 1, 0}, v.edgeIds)
			assert.Equal(t, []int{5, 4, 3, 2, 1, 0}, v.baseNodes)
			assert.Equal(t, []int{6, 5, 4, 3, 2, 1}, v.adjNodes)
			if edgeBased {
				assert.Equal(t, []int{4, 3, 2, 1, 0, prevEdge}, v.prevOrNextEdgeIds)
			}

			// backward, normal order
			v = newTestVisitor(f.chGraph)
			f.visitBwd(10, 0, false, v)
			assert.Equal(t, []int{5, 4, 3, 2, 1, 0}, v.edgeIds)
			assert.Equal(t, []int{6, 5, 4, 3, 2, 1}, v.baseNodes)
			assert.Equal(t, []int{5, 4, 3, 2, 1, 0}, v.adjNodes)
			if edgeBased {
				assert.Equal(t, []int{nextEdge, 5, 4, 3, 2, 1}, v.prevOrNextEdgeIds)
			}

			// backward, reverse order
			v = newTestVisitor(f.chGraph)
			f.visitBwd(10, 0, true, v)
			assert.Equal(t, []int{0, 1, 2, 3, 4, 5}, v.edgeIds)
			assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, v.baseNodes)
			assert.Equal(t, []int{0, 1, 2, 3, 4, 5}, v.adjNodes)
			if edgeBased {
				assert.Equal(t, []int{1, 2, 3, 4, 5, nextEdge}, v.prevOrNextEdgeIds)
			}
		})
	}
}

func TestShortcutUnpacker_LoopShortcut(t *testing.T) {
	// loop shortcuts only exist for edge-based CH
	f := newUnpackFixture(true)
	//     3
	//    / \
	//   2   4
	//    \ /
	// 0 - 1 - 5
	f.graph.Edge(0, 1).SetDistance(1).SetDecimalBothDir(f.speedEnc, 20, 10)
	f.graph.Edge(1, 2).SetDistance(1).SetDecimalBothDir(f.speedEnc, 20, 10)
	f.graph.Edge(2, 3).SetDistance(1).SetDecimalBothDir(f.speedEnc, 20, 10)
	f.graph.Edge(3, 4).SetDistance(1).SetDecimalBothDir(f.speedEnc, 20, 10)
	f.graph.Edge(4, 1).SetDistance(1).SetDecimalBothDir(f.speedEnc, 20, 10)
	f.graph.Edge(1, 5).SetDistance(1).SetDecimalBothDir(f.speedEnc, 20, 10)
	f.freeze()

	f.setCHLevels(2, 4, 3, 1, 5, 0)
	f.shortcut(3, 1, 1, 2, 2, 4, true)
	f.shortcut(3, 1, 3, 4, 6, 8, false)
	f.shortcut(1, 1, 6, 7, 2, 8, false)
	f.shortcut(1, 0, 0, 8, 0, 8, true)
	f.shortcut(5, 0, 9, 5, 0, 10, true)

	// forward, normal order
	v := newTestVisitor(f.chGraph)
	f.visitFwd(10, 5, false, v)
	assert.Equal(t, []int{0, 1, 2, 3, 4, 5}, v.edgeIds)
	assert.Equal(t, []int{0, 1, 2, 3, 4, 1}, v.baseNodes)
	assert.Equal(t, []int{1, 2, 3, 4, 1, 5}, v.adjNodes)
	assert.Equal(t, []int{prevEdge, 0, 1, 2, 3, 4}, v.prevOrNextEdgeIds)

	// forward, reverse order
	v = newTestVisitor(f.chGraph)
	f.visitFwd(10, 5, true, v)
	assert.Equal(t, []int{5, 4, 3, 2, 1, 0}, v.edgeIds)
	assert.Equal(t, []int{1, 4, 3, 2, 1, 0}, v.baseNodes)
	assert.Equal(t, []int{5, 1, 4, 3, 2, 1}, v.adjNodes)
	assert.Equal(t, []int{4, 3, 2, 1, 0, prevEdge}, v.prevOrNextEdgeIds)

	// backward, normal order
	v = newTestVisitor(f.chGraph)
	f.visitBwd(10, 0, false, v)
	assert.Equal(t, []int{5, 4, 3, 2, 1, 0}, v.edgeIds)
	assert.Equal(t, []int{5, 1, 4, 3, 2, 1}, v.baseNodes)
	assert.Equal(t, []int{1, 4, 3, 2, 1, 0}, v.adjNodes)
	assert.Equal(t, []int{nextEdge, 5, 4, 3, 2, 1}, v.prevOrNextEdgeIds)

	// backward, reverse order
	v = newTestVisitor(f.chGraph)
	f.visitBwd(10, 0, true, v)
	assert.Equal(t, []int{0, 1, 2, 3, 4, 5}, v.edgeIds)
	assert.Equal(t, []int{1, 2, 3, 4, 1, 5}, v.baseNodes)
	assert.Equal(t, []int{0, 1, 2, 3, 4, 1}, v.adjNodes)
	assert.Equal(t, []int{1, 2, 3, 4, 5, nextEdge}, v.prevOrNextEdgeIds)
}

func TestShortcutUnpacker_WithCalcTurnWeight(t *testing.T) {
	// edge-based only
	f := newUnpackFixture(true)
	//      2 5 3 2 1 4 6      turn costs ->
	// prev 0-1-2-3-4-5-6 next
	//      1 0 1 4 2 3 2      turn costs <-
	edge0 := f.graph.Edge(0, 1).SetDistance(1).SetDecimalBothDir(f.speedEnc, 20, 10)
	edge1 := f.graph.Edge(1, 2).SetDistance(1).SetDecimalBothDir(f.speedEnc, 20, 10)
	edge2 := f.graph.Edge(2, 3).SetDistance(1).SetDecimalBothDir(f.speedEnc, 20, 10)
	edge3 := f.graph.Edge(3, 4).SetDistance(1).SetDecimalBothDir(f.speedEnc, 20, 10)
	edge4 := f.graph.Edge(4, 5).SetDistance(1).SetDecimalBothDir(f.speedEnc, 20, 10)
	edge5 := f.graph.Edge(5, 6).SetDistance(1).SetDecimalBothDir(f.speedEnc, 20, 10)
	f.freeze()

	// turn costs ->
	f.setTurnCost(prevEdge, 0, edge0.GetEdge(), 2.0)
	f.setTurnCost(edge0.GetEdge(), 1, edge1.GetEdge(), 5.0)
	f.setTurnCost(edge1.GetEdge(), 2, edge2.GetEdge(), 3.0)
	f.setTurnCost(edge2.GetEdge(), 3, edge3.GetEdge(), 2.0)
	f.setTurnCost(edge3.GetEdge(), 4, edge4.GetEdge(), 1.0)
	f.setTurnCost(edge4.GetEdge(), 5, edge5.GetEdge(), 4.0)
	f.setTurnCost(edge5.GetEdge(), 6, nextEdge, 6.0)
	// turn costs <-
	f.setTurnCost(nextEdge, 6, edge5.GetEdge(), 2.0)
	f.setTurnCost(edge5.GetEdge(), 5, edge4.GetEdge(), 3.0)
	f.setTurnCost(edge4.GetEdge(), 4, edge3.GetEdge(), 2.0)
	f.setTurnCost(edge3.GetEdge(), 3, edge2.GetEdge(), 4.0)
	f.setTurnCost(edge2.GetEdge(), 2, edge1.GetEdge(), 1.0)
	f.setTurnCost(edge1.GetEdge(), 1, edge0.GetEdge(), 0.0)
	f.setTurnCost(edge0.GetEdge(), 0, prevEdge, 1.0)

	f.setCHLevels(1, 3, 5, 4, 2, 0, 6)
	f.shortcut(4, 2, 2, 3, 4, 6, true)
	f.shortcut(4, 6, 4, 5, 8, 10, false)
	f.shortcut(2, 0, 0, 1, 0, 2, true)
	f.shortcut(2, 6, 6, 7, 4, 10, false)
	f.shortcut(0, 6, 8, 9, 0, 10, false)

	// forward, normal order
	v := newTurnWeightingVisitor(f.chGraph)
	f.visitFwd(10, 6, false, v)
	assert.InDelta(t, 6*0.05+17, v.weight, 1e-3, "wrong weight")
	assert.Equal(t, int64(6*50+17000), v.time, "wrong time")

	// forward, reverse order
	v = newTurnWeightingVisitor(f.chGraph)
	f.visitFwd(10, 6, true, v)
	assert.InDelta(t, 6*0.05+17, v.weight, 1e-3, "wrong weight")
	assert.Equal(t, int64(6*50+17000), v.time, "wrong time")

	// backward, normal order
	v = newTurnWeightingVisitor(f.chGraph)
	f.visitBwd(10, 0, false, v)
	assert.InDelta(t, 6*0.05+21, v.weight, 1e-3, "wrong weight")
	assert.Equal(t, int64(6*50+21000), v.time, "wrong time")

	// backward, reverse order
	v = newTurnWeightingVisitor(f.chGraph)
	f.visitBwd(10, 0, true, v)
	assert.InDelta(t, 6*0.05+21, v.weight, 1e-3, "wrong weight")
	assert.Equal(t, int64(6*50+21000), v.time, "wrong time")
}
