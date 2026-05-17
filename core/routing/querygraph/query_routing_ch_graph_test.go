package querygraph

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/storage/index"
	"gohopper/core/util"
)

// Shortcut direction flags (local copies to avoid import cycle with
// gohopper/core/routing/ch which transitively depends on this package).
const (
	testScFwdDir  = 0x1
	testScDirMask = 0x3
)

type queryRoutingCHGraphFixture struct {
	speedEnc    ev.DecimalEncodedValue
	turnCostEnc ev.DecimalEncodedValue
	weighting   *weighting.SpeedWeighting
	graph       *storage.BaseGraph
	na          storage.NodeAccess
}

func newQueryRoutingCHGraphFixture() *queryRoutingCHGraphFixture {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	turnCostEnc := ev.TurnCostCreate("car", 5)
	em := routingutil.Start().Add(speedEnc).AddTurnCostEncodedValue(turnCostEnc).Build()
	graph := storage.NewBaseGraphBuilder(em.BytesForFlags).SetWithTurnCosts(true).CreateGraph()
	w := weighting.NewSpeedWeightingWithTurnCosts(speedEnc, turnCostEnc, graph.GetTurnCostStorage(), graph.GetNodeAccess(), math.Inf(1))
	return &queryRoutingCHGraphFixture{
		speedEnc:    speedEnc,
		turnCostEnc: turnCostEnc,
		weighting:   w,
		graph:       graph,
		na:          graph.GetNodeAccess(),
	}
}

func (f *queryRoutingCHGraphFixture) edgeBasedCHGraph() (*storage.RoutingCHGraphImpl, *storage.CHStorageBuilder) {
	store := storage.CHStorageFromGraph(f.graph, "x", true)
	chBuilder := storage.NewCHStorageBuilder(store)
	chGraph := storage.NewRoutingCHGraph(f.graph, store, f.weighting)
	return chGraph, chBuilder
}

func (f *queryRoutingCHGraphFixture) addEdge(from, to int) util.EdgeIteratorState {
	dist := util.DistPlane.CalcDist(
		f.na.GetLat(from), f.na.GetLon(from),
		f.na.GetLat(to), f.na.GetLon(to),
	)
	return f.graph.Edge(from, to).SetDistance(dist).SetDecimal(f.speedEnc, 10).SetReverseDecimal(f.speedEnc, 10)
}

func snapForEdge(t *testing.T, lat, lon float64, edge util.EdgeIteratorState) *index.Snap {
	t.Helper()
	s := index.NewSnap(lat, lon)
	s.SetClosestEdge(edge)
	s.SetWayIndex(0)
	s.SetSnappedPosition(index.Edge)
	s.CalcSnappedPoint(util.DistPlane)
	return s
}

// --- assertion helpers (ported from Java) ---

func assertNextEdge(t *testing.T, iter storage.RoutingCHEdgeIterator, base, adj, origEdge int) {
	t.Helper()
	require.True(t, iter.Next(), "there is no further edge")
	assertEdgeState(t, iter, base, adj, origEdge)
}

func assertEdgeState(t *testing.T, edgeState storage.RoutingCHEdgeIteratorState, base, adj, origEdge int) {
	t.Helper()
	assert.False(t, edgeState.IsShortcut(), "did not expect a shortcut")
	assert.Equal(t, base, edgeState.GetBaseNode(), "wrong base node")
	assert.Equal(t, adj, edgeState.GetAdjNode(), "wrong adj node")
	assert.Equal(t, origEdge, edgeState.GetOrigEdge(), "wrong orig edge")
}

func assertNextShortcut(t *testing.T, iter storage.RoutingCHEdgeIterator, base, adj, skip1, skip2 int) {
	t.Helper()
	require.True(t, iter.Next(), "there is no further edge")
	assertShortcut(t, iter, base, adj, skip1, skip2)
}

func assertShortcut(t *testing.T, edgeState storage.RoutingCHEdgeIteratorState, base, adj, skip1, skip2 int) {
	t.Helper()
	assert.True(t, edgeState.IsShortcut(), "expected a shortcut")
	assert.Equal(t, base, edgeState.GetBaseNode(), "wrong base node")
	assert.Equal(t, adj, edgeState.GetAdjNode(), "wrong adj node")
	assert.Equal(t, util.NoEdge, edgeState.GetOrigEdge(), "wrong orig edge")
	assert.Equal(t, skip1, edgeState.GetSkippedEdge1(), "wrong skip1 edge")
	assert.Equal(t, skip2, edgeState.GetSkippedEdge2(), "wrong skip2 edge")
}

func assertEnd(t *testing.T, iter storage.RoutingCHEdgeIterator) {
	t.Helper()
	assert.False(t, iter.Next())
}

func chEdgeBetween(explorer storage.RoutingCHEdgeExplorer, base, adj int) int {
	iter := explorer.SetBaseNode(base)
	for iter.Next() {
		if iter.GetAdjNode() == adj {
			return iter.GetEdge()
		}
	}
	return util.NoEdge
}

func assertEdgeAtNodes(t *testing.T, g storage.RoutingCHGraph, edge, p, q int) {
	t.Helper()
	fails := true
	iter := g.CreateOutEdgeExplorer().SetBaseNode(p)
	for iter.Next() {
		if iter.GetAdjNode() == q && iter.GetEdge() == edge {
			fails = false
		}
	}
	iter = g.CreateInEdgeExplorer().SetBaseNode(q)
	for iter.Next() {
		if iter.GetAdjNode() == p && iter.GetEdge() == edge {
			fails = false
		}
	}
	assert.False(t, fails)
}

func assertNodesConnected(t *testing.T, g storage.RoutingCHGraph, p, q int, bothDirections bool) {
	t.Helper()
	chEdge := chEdgeBetween(g.CreateOutEdgeExplorer(), p, q)
	require.NotEqual(t, util.NoEdge, chEdge, "No CH out-edge %d->%d", p, q)
	assertEdgeAtNodes(t, g, chEdge, p, q)
	chEdge = chEdgeBetween(g.CreateInEdgeExplorer(), q, p)
	require.NotEqual(t, util.NoEdge, chEdge, "No CH in-edge %d<-%d", q, p)
	assertEdgeAtNodes(t, g, chEdge, p, q)

	revCHEdge := chEdgeBetween(g.CreateOutEdgeExplorer(), q, p)
	if bothDirections {
		require.NotEqual(t, util.NoEdge, revCHEdge, "No CH out-edge %d->%d", q, p)
		assertEdgeAtNodes(t, g, revCHEdge, p, q)
	} else {
		require.Equal(t, util.NoEdge, revCHEdge, "Unexpected CH out-edge %d->%d", q, p)
	}
	revCHEdge = chEdgeBetween(g.CreateInEdgeExplorer(), p, q)
	if bothDirections {
		require.NotEqual(t, util.NoEdge, revCHEdge, "No CH in-edge %d<-%d", q, p)
		assertEdgeAtNodes(t, g, revCHEdge, p, q)
	} else {
		require.Equal(t, util.NoEdge, revCHEdge, "Unexpected CH in-edge %d<-%d", q, p)
	}
}

func assertGetEdgeIteratorState(t *testing.T, g storage.RoutingCHGraph, base, adj, origEdge int) {
	t.Helper()
	chEdge := chEdgeBetween(g.CreateOutEdgeExplorer(), base, adj)
	assertEdgeState(t, g.GetEdgeIteratorState(chEdge, adj), base, adj, origEdge)
	assertEdgeState(t, g.GetEdgeIteratorState(chEdge, base), adj, base, origEdge)
}

func assertGetEdgeIteratorShortcut(t *testing.T, g storage.RoutingCHGraph, base, adj, skip1, skip2 int) {
	t.Helper()
	chEdge := chEdgeBetween(g.CreateOutEdgeExplorer(), base, adj)
	assertShortcut(t, g.GetEdgeIteratorState(chEdge, adj), base, adj, skip1, skip2)
	assertShortcut(t, g.GetEdgeIteratorState(chEdge, base), adj, base, skip1, skip2)
}

// --- tests ported from QueryRoutingCHGraphTest.java ---

func TestQueryRoutingCHGraph_Basic(t *testing.T) {
	// 0-1-2
	f := newQueryRoutingCHGraphFixture()
	f.graph.Edge(0, 1).SetDistance(10).SetDecimal(f.speedEnc, 10).SetReverseDecimal(f.speedEnc, 10)
	f.graph.Edge(1, 2).SetDistance(10).SetDecimal(f.speedEnc, 10).SetReverseDecimal(f.speedEnc, 10)
	f.graph.Freeze()
	assert.Equal(t, 2, f.graph.GetEdges())

	chGraph, _ := f.edgeBasedCHGraph()

	qg := CreateFromSnaps(f.graph, nil)
	qch := NewQueryRoutingCHGraph(chGraph, qg)

	assert.Equal(t, 3, qch.GetNodes())
	assert.Equal(t, 2, qch.GetEdges())
	assert.True(t, qch.IsEdgeBased())
	assert.True(t, qch.HasTurnCosts())

	assertNodesConnected(t, qch, 0, 1, true)
	assertNodesConnected(t, qch, 1, 2, true)

	outIter := qch.CreateOutEdgeExplorer().SetBaseNode(0)
	assertNextEdge(t, outIter, 0, 1, 0)
	assertEnd(t, outIter)

	inIter := qch.CreateInEdgeExplorer().SetBaseNode(1)
	assertNextEdge(t, inIter, 1, 2, 1)
	assertNextEdge(t, inIter, 1, 0, 0)
	assertEnd(t, inIter)

	inIter = qch.CreateInEdgeExplorer().SetBaseNode(2)
	assertNextEdge(t, inIter, 2, 1, 1)
	assertEnd(t, inIter)
}

func TestQueryRoutingCHGraph_WithShortcuts(t *testing.T) {
	// 0-1-2
	//  \-/
	f := newQueryRoutingCHGraphFixture()
	f.graph.Edge(0, 1).SetDistance(10).SetDecimal(f.speedEnc, 10).SetReverseDecimal(f.speedEnc, 10)
	f.graph.Edge(1, 2).SetDistance(10).SetDecimal(f.speedEnc, 10).SetReverseDecimal(f.speedEnc, 10)
	f.graph.Freeze()
	assert.Equal(t, 2, f.graph.GetEdges())

	chGraph, chBuilder := f.edgeBasedCHGraph()
	chBuilder.SetIdentityLevels()
	chBuilder.AddShortcutEdgeBased(0, 2, testScFwdDir, 20, 0, 1, 0, 2)

	qg := CreateFromSnaps(f.graph, nil)
	qch := NewQueryRoutingCHGraph(chGraph, qg)

	assert.Equal(t, 3, qch.GetNodes())
	assert.Equal(t, 3, qch.GetEdges())

	assertNodesConnected(t, qch, 0, 1, true)
	assertNodesConnected(t, qch, 1, 2, true)

	outIter := qch.CreateOutEdgeExplorer().SetBaseNode(0)
	assertNextShortcut(t, outIter, 0, 2, 0, 1)
	assertNextEdge(t, outIter, 0, 1, 0)
	assertEnd(t, outIter)

	inIter := qch.CreateInEdgeExplorer().SetBaseNode(2)
	assertNextEdge(t, inIter, 2, 1, 1)
	assertEnd(t, inIter)
}

func TestQueryRoutingCHGraph_WithVirtualEdges(t *testing.T) {
	//  2 3
	// 0-x-1-2
	//   3
	f := newQueryRoutingCHGraphFixture()
	f.na.SetNode(0, 50.00, 10.00, math.NaN())
	f.na.SetNode(1, 50.00, 10.10, math.NaN())
	f.na.SetNode(2, 50.00, 10.20, math.NaN())
	edge := f.addEdge(0, 1)
	f.addEdge(1, 2)
	f.graph.Freeze()
	assert.Equal(t, 2, f.graph.GetEdges())

	chGraph, _ := f.edgeBasedCHGraph()

	snap := snapForEdge(t, 50.00, 10.05, edge)

	qg := CreateFromSnaps(f.graph, []*index.Snap{snap})
	qch := NewQueryRoutingCHGraph(chGraph, qg)

	assert.Equal(t, 4, qch.GetNodes())
	assert.Equal(t, 2+4, qch.GetEdges())

	assertNodesConnected(t, qch, 1, 2, true)
	// virtual edges at virtual node 3
	assertNodesConnected(t, qch, 0, 3, true)
	assertNodesConnected(t, qch, 3, 1, true)

	// out-iter at real node
	outIter := qch.CreateOutEdgeExplorer().SetBaseNode(2)
	assertNextEdge(t, outIter, 2, 1, 1)
	assertEnd(t, outIter)

	// in-iter at real node
	inIter := qch.CreateInEdgeExplorer().SetBaseNode(2)
	assertNextEdge(t, inIter, 2, 1, 1)
	assertEnd(t, inIter)

	// out-iter at real node next to virtual node
	outIter = qch.CreateOutEdgeExplorer().SetBaseNode(0)
	assertNextEdge(t, outIter, 0, 3, 2)
	assertEnd(t, outIter)

	// in-iter at real node next to virtual node
	inIter = qch.CreateInEdgeExplorer().SetBaseNode(1)
	assertNextEdge(t, inIter, 1, 3, 3)
	assertNextEdge(t, inIter, 1, 2, 1)
	assertEnd(t, inIter)

	// out-iter at virtual node
	outIter = qch.CreateOutEdgeExplorer().SetBaseNode(3)
	assertNextEdge(t, outIter, 3, 0, 2)
	assertNextEdge(t, outIter, 3, 1, 3)
	assertEnd(t, outIter)

	// in-iter at virtual node
	inIter = qch.CreateInEdgeExplorer().SetBaseNode(3)
	assertNextEdge(t, inIter, 3, 0, 2)
	assertNextEdge(t, inIter, 3, 1, 3)
	assertEnd(t, inIter)
}

func TestQueryRoutingCHGraph_WithVirtualEdgesAndShortcuts(t *testing.T) {
	//  /---\
	// 0-x-1-2
	//   3
	f := newQueryRoutingCHGraphFixture()
	f.na.SetNode(0, 50.00, 10.00, math.NaN())
	f.na.SetNode(1, 50.00, 10.10, math.NaN())
	f.na.SetNode(2, 50.00, 10.20, math.NaN())
	edge := f.addEdge(0, 1)
	f.addEdge(1, 2)
	f.graph.Freeze()
	assert.Equal(t, 2, f.graph.GetEdges())

	chGraph, chBuilder := f.edgeBasedCHGraph()
	chBuilder.SetIdentityLevels()
	chBuilder.AddShortcutEdgeBased(0, 2, testScFwdDir, 20, 0, 1, 0, 2)

	snap := snapForEdge(t, 50.00, 10.05, edge)

	qg := CreateFromSnaps(f.graph, []*index.Snap{snap})
	qch := NewQueryRoutingCHGraph(chGraph, qg)

	assert.Equal(t, 4, qch.GetNodes())
	assert.Equal(t, 3+4, qch.GetEdges())

	assertNodesConnected(t, qch, 0, 3, true)
	assertNodesConnected(t, qch, 3, 1, true)
	assertNodesConnected(t, qch, 1, 2, true)

	// at real nodes
	outIter := qch.CreateOutEdgeExplorer().SetBaseNode(0)
	// note that orig edge of virtual edges corresponds to the id of the virtual edge on the base graph
	assertNextEdge(t, outIter, 0, 3, 2)
	assertNextShortcut(t, outIter, 0, 2, 0, 1)
	assertEnd(t, outIter)

	inIter := qch.CreateInEdgeExplorer().SetBaseNode(2)
	assertNextEdge(t, inIter, 2, 1, 1)
	assertEnd(t, inIter)

	// at virtual nodes
	outIter = qch.CreateOutEdgeExplorer().SetBaseNode(3)
	assertNextEdge(t, outIter, 3, 0, 2)
	assertNextEdge(t, outIter, 3, 1, 3)
	assertEnd(t, outIter)

	inIter = qch.CreateInEdgeExplorer().SetBaseNode(3)
	assertNextEdge(t, inIter, 3, 0, 2)
	assertNextEdge(t, inIter, 3, 1, 3)
	assertEnd(t, inIter)
}

func TestQueryRoutingCHGraph_GetBaseGraph(t *testing.T) {
	f := newQueryRoutingCHGraphFixture()
	f.graph.Edge(0, 1).SetDistance(10).SetDecimal(f.speedEnc, 10).SetReverseDecimal(f.speedEnc, 10)
	f.graph.Freeze()

	chGraph, _ := f.edgeBasedCHGraph()

	qg := CreateFromSnaps(f.graph, nil)
	assert.Same(t, f.graph.GetBaseGraph(), chGraph.GetBaseGraph())
	qch := NewQueryRoutingCHGraph(chGraph, qg)
	assert.Same(t, qg, qch.GetBaseGraph())
}

func TestQueryRoutingCHGraph_GetEdgeIteratorState(t *testing.T) {
	//  /---\
	// 0-x-1-2
	//   3
	f := newQueryRoutingCHGraphFixture()
	f.na.SetNode(0, 50.00, 10.00, math.NaN())
	f.na.SetNode(1, 50.00, 10.10, math.NaN())
	f.na.SetNode(2, 50.00, 10.20, math.NaN())
	edge := f.addEdge(0, 1)
	f.addEdge(1, 2)
	f.graph.Freeze()

	chGraph, chBuilder := f.edgeBasedCHGraph()
	chBuilder.SetIdentityLevels()
	chBuilder.AddShortcutEdgeBased(0, 2, testScFwdDir, 20, 0, 1, 0, 2)

	snap := snapForEdge(t, 50.00, 10.05, edge)

	qg := CreateFromSnaps(f.graph, []*index.Snap{snap})
	qch := NewQueryRoutingCHGraph(chGraph, qg)

	assertGetEdgeIteratorState(t, qch, 1, 2, 1)
	assertGetEdgeIteratorShortcut(t, qch, 0, 2, 0, 1)
	// the orig edge corresponds to the edge id of the edge in the (base) query graph
	assertGetEdgeIteratorState(t, qch, 0, 3, 2)
	assertGetEdgeIteratorState(t, qch, 3, 0, 2)
	assertGetEdgeIteratorState(t, qch, 1, 3, 3)
	assertGetEdgeIteratorState(t, qch, 3, 1, 3)
}

func TestQueryRoutingCHGraph_GetWeighting(t *testing.T) {
	f := newQueryRoutingCHGraphFixture()
	f.graph.Freeze()
	qg := CreateFromSnaps(f.graph, nil)

	chGraph, _ := f.edgeBasedCHGraph()

	qch := NewQueryRoutingCHGraph(chGraph, qg)
	// maybe query CH graph should return query graph weighting instead?
	assert.Same(t, f.weighting, qch.GetWeighting())
}

func TestQueryRoutingCHGraph_GetLevel(t *testing.T) {
	// 0-x-1
	f := newQueryRoutingCHGraphFixture()
	f.na.SetNode(0, 50.00, 10.00, math.NaN())
	f.na.SetNode(1, 50.00, 10.10, math.NaN())
	edge := f.addEdge(0, 1)
	f.graph.Freeze()

	chGraph, chBuilder := f.edgeBasedCHGraph()
	chBuilder.SetLevel(0, 5)
	chBuilder.SetLevel(1, 7)

	snap := snapForEdge(t, 50.00, 10.05, edge)

	qg := CreateFromSnaps(f.graph, []*index.Snap{snap})
	qch := NewQueryRoutingCHGraph(chGraph, qg)
	assert.Equal(t, 5, qch.GetLevel(0))
	assert.Equal(t, 7, qch.GetLevel(1))
	assert.Equal(t, math.MaxInt, qch.GetLevel(2))
}

func TestQueryRoutingCHGraph_GetWeight(t *testing.T) {
	//  /---\
	// 0-x-1-2
	//   3
	f := newQueryRoutingCHGraphFixture()
	f.na.SetNode(0, 50.00, 10.00, math.NaN())
	f.na.SetNode(1, 50.00, 10.10, math.NaN())
	f.na.SetNode(2, 50.00, 10.20, math.NaN())
	// use different speeds for the two directions
	edge := f.addEdge(0, 1).SetDecimal(f.speedEnc, 30).SetReverseDecimal(f.speedEnc, 10)
	f.addEdge(1, 2)
	f.graph.Freeze()

	chGraph, chBuilder := f.edgeBasedCHGraph()
	chBuilder.SetIdentityLevels()
	chBuilder.AddShortcutEdgeBased(0, 2, testScDirMask, 20, 0, 1, 0, 2)

	// without query graph
	iter := chGraph.CreateOutEdgeExplorer().SetBaseNode(0)
	assertNextShortcut(t, iter, 0, 2, 0, 1)
	assert.InDelta(t, 20, iter.GetWeight(false), 1e-6)
	assert.InDelta(t, 20, iter.GetWeight(true), 1e-6)
	assertNextEdge(t, iter, 0, 1, 0)
	assert.InDelta(t, 238.249066, iter.GetWeight(false), 1e-6)
	assert.InDelta(t, 714.7472, iter.GetWeight(true), 1e-6)
	assertEnd(t, iter)

	// for incoming edges it's the same
	iter = chGraph.CreateInEdgeExplorer().SetBaseNode(0)
	assertNextShortcut(t, iter, 0, 2, 0, 1)
	assert.InDelta(t, 20, iter.GetWeight(false), 1e-6)
	assert.InDelta(t, 20, iter.GetWeight(true), 1e-6)
	assertNextEdge(t, iter, 0, 1, 0)
	assert.InDelta(t, 238.249066, iter.GetWeight(false), 1e-6)
	assert.InDelta(t, 714.7472, iter.GetWeight(true), 1e-6)
	assertEnd(t, iter)

	// now including virtual edges
	snap := snapForEdge(t, 50.00, 10.05, edge)

	qg := CreateFromSnaps(f.graph, []*index.Snap{snap})
	qch := NewQueryRoutingCHGraph(chGraph, qg)

	iter = qch.CreateOutEdgeExplorer().SetBaseNode(0)
	assertNextEdge(t, iter, 0, 3, 2)
	// should be about half the weight as for the original edge as the query point is in the middle of the edge
	assert.InDelta(t, 119.12453, iter.GetWeight(false), 1e-4)
	assert.InDelta(t, 357.373605, iter.GetWeight(true), 1e-4)
	assertNextShortcut(t, iter, 0, 2, 0, 1)
	assert.InDelta(t, 20, iter.GetWeight(false), 1e-6)
	assert.InDelta(t, 20, iter.GetWeight(true), 1e-6)
	assertEnd(t, iter)

	iter = qch.CreateInEdgeExplorer().SetBaseNode(0)
	assertNextEdge(t, iter, 0, 3, 2)
	assert.InDelta(t, 119.12453, iter.GetWeight(false), 1e-4)
	assert.InDelta(t, 357.373605, iter.GetWeight(true), 1e-4)
	assertNextShortcut(t, iter, 0, 2, 0, 1)
	assert.InDelta(t, 20, iter.GetWeight(false), 1e-6)
	assert.InDelta(t, 20, iter.GetWeight(true), 1e-6)
	assertEnd(t, iter)

	// at the virtual node
	iter = qch.CreateOutEdgeExplorer().SetBaseNode(3)
	assertNextEdge(t, iter, 3, 0, 2)
	assert.InDelta(t, 357.373605, iter.GetWeight(false), 1e-4)
	assert.InDelta(t, 119.12453, iter.GetWeight(true), 1e-4)
	assertNextEdge(t, iter, 3, 1, 3)
	assert.InDelta(t, 119.12453, iter.GetWeight(false), 1e-4)
	assert.InDelta(t, 357.373605, iter.GetWeight(true), 1e-4)
	assertEnd(t, iter)

	iter = qch.CreateInEdgeExplorer().SetBaseNode(3)
	assertNextEdge(t, iter, 3, 0, 2)
	assert.InDelta(t, 357.373605, iter.GetWeight(false), 1e-4)
	assert.InDelta(t, 119.12453, iter.GetWeight(true), 1e-4)
	assertNextEdge(t, iter, 3, 1, 3)
	assert.InDelta(t, 119.12453, iter.GetWeight(false), 1e-4)
	assert.InDelta(t, 357.373605, iter.GetWeight(true), 1e-4)
	assertEnd(t, iter)

	// getting a single edge
	edgeState := qch.GetEdgeIteratorState(3, 3)
	assertEdgeState(t, edgeState, 0, 3, 2)
	assert.InDelta(t, 119.12453, edgeState.GetWeight(false), 1e-4)
	assert.InDelta(t, 357.373605, edgeState.GetWeight(true), 1e-4)

	edgeState = qch.GetEdgeIteratorState(3, 0)
	assertEdgeState(t, edgeState, 3, 0, 2)
	assert.InDelta(t, 357.373605, edgeState.GetWeight(false), 1e-4)
	assert.InDelta(t, 119.12453, edgeState.GetWeight(true), 1e-4)
}

func TestQueryRoutingCHGraph_GetTurnCost(t *testing.T) {
	//  /-----\
	// 0-x-1-x-2
	//   3   4
	f := newQueryRoutingCHGraphFixture()
	f.na.SetNode(0, 50.00, 10.00, math.NaN())
	f.na.SetNode(1, 50.00, 10.10, math.NaN())
	f.na.SetNode(2, 50.00, 10.20, math.NaN())
	edge1 := f.addEdge(0, 1)
	edge2 := f.addEdge(1, 2)
	f.graph.GetTurnCostStorage().SetDecimal(f.na, f.turnCostEnc, 0, 1, 1, 5)
	f.graph.Freeze()

	chGraph, chBuilder := f.edgeBasedCHGraph()
	chBuilder.SetIdentityLevels()
	chBuilder.AddShortcutEdgeBased(0, 2, testScFwdDir, 20, 0, 1, 0, 2)

	// without virtual nodes
	assert.InDelta(t, 5, chGraph.GetTurnWeight(0, 1, 1), 1e-9)

	// with virtual nodes
	snap1 := snapForEdge(t, 50.00, 10.05, edge1)
	snap2 := snapForEdge(t, 50.00, 10.15, edge2)

	qg := CreateFromSnaps(f.graph, []*index.Snap{snap1, snap2})
	qch := NewQueryRoutingCHGraph(chGraph, qg)
	assert.InDelta(t, 5, qch.GetTurnWeight(0, 1, 1), 1e-9)

	// take a look at edges 3->1 and 1->4, their original edge ids are 3 and 4 (not 4 and 5)
	assertNodesConnected(t, qch, 3, 1, true)
	assertNodesConnected(t, qch, 1, 4, true)
	expectedEdge31 := 3
	expectedEdge14 := 4
	iter := qch.CreateOutEdgeExplorer().SetBaseNode(3)
	assertNextEdge(t, iter, 3, 0, 2)
	assertNextEdge(t, iter, 3, 1, expectedEdge31)
	assertEnd(t, iter)

	iter = qch.CreateOutEdgeExplorer().SetBaseNode(1)
	assertNextEdge(t, iter, 1, 3, 3)
	assertNextEdge(t, iter, 1, 4, expectedEdge14)
	assertEnd(t, iter)

	// check the turn weight between these edges
	assert.InDelta(t, 5, qch.GetTurnWeight(expectedEdge31, 1, expectedEdge14), 1e-9)
}
