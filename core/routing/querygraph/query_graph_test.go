package querygraph

import (
	"math"
	"testing"

	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/storage/index"
	"gohopper/core/util"
)

var (
	speedEnc       *ev.DecimalEncodedValueImpl
	encodingManager *routingutil.EncodingManager
)

func setupTest(t *testing.T) *storage.BaseGraph {
	t.Helper()
	speedEnc = ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	encodingManager = routingutil.Start().Add(speedEnc).Build()
	g := storage.NewBaseGraphBuilder(encodingManager.BytesForFlags).CreateGraph()
	t.Cleanup(func() { g.Close() })
	return g
}

func initGraph(g *storage.BaseGraph) {
	//
	//  /*-*\
	// 0     1
	// |
	// 2
	na := g.GetNodeAccess()
	na.SetNode(0, 1, 0, math.NaN())
	na.SetNode(1, 1, 2.5, math.NaN())
	na.SetNode(2, 0, 0, math.NaN())
	g.Edge(0, 2).SetDistance(10).SetDecimal(speedEnc, 60).SetReverseDecimal(speedEnc, 60)
	g.Edge(0, 1).SetDistance(10).SetDecimal(speedEnc, 60).SetReverseDecimal(speedEnc, 60).
		SetWayGeometry(util.CreatePointList(1.5, 1, 1.5, 1.5))
}

func createLocationResult(lat, lon float64, edge util.EdgeIteratorState, wayIndex int, pos index.Position) *index.Snap {
	if edge == nil {
		panic("specify edge != nil")
	}
	tmp := index.NewSnap(lat, lon)
	tmp.SetClosestEdge(edge)
	tmp.SetWayIndex(wayIndex)
	tmp.SetSnappedPosition(pos)
	tmp.CalcSnappedPoint(util.DistEarth)
	return tmp
}

func getPoints(g storage.Graph, base, adj int) *util.PointList {
	edge := getEdge(g, base, adj)
	if edge == nil {
		panic("edge not found")
	}
	return edge.FetchWayGeometry(util.FetchModeAll)
}

func getEdge(g storage.Graph, base, adj int) util.EdgeIteratorState {
	explorer := g.CreateEdgeExplorer(routingutil.AllEdges)
	count := util.CountAdj(explorer.SetBaseNode(base), adj)
	if count > 1 {
		panic("there are multiple edges")
	}
	if count == 0 {
		return nil
	}
	iter := explorer.SetBaseNode(base)
	for iter.Next() {
		if iter.GetAdjNode() == adj {
			return iter
		}
	}
	panic("should not happen")
}

func lookup(g *storage.BaseGraph, snaps ...*index.Snap) *QueryGraph {
	return CreateFromSnaps(g, snaps)
}

func TestOneVirtualNode(t *testing.T) {
	g := setupTest(t)
	initGraph(g)
	expl := g.CreateEdgeExplorer(routingutil.AllEdges)

	// snap directly to tower node
	// a)
	iter := expl.SetBaseNode(2)
	iter.Next()

	res := createLocationResult(1, -1, iter, 0, index.Tower)
	_ = lookup(g, res)
	assertGHPointEqual(t, util.GHPoint{Lat: 0, Lon: 0}, res.GetSnappedPoint().GHPoint)

	// b)
	res = createLocationResult(1, -1, iter, 1, index.Tower)
	_ = lookup(g, res)
	assertGHPointEqual(t, util.GHPoint{Lat: 1, Lon: 0}, res.GetSnappedPoint().GHPoint)

	// c)
	iter = expl.SetBaseNode(1)
	iter.Next()
	res = createLocationResult(1.2, 2.7, iter, 0, index.Tower)
	queryGraph2 := lookup(g, res)
	assertGHPointEqual(t, util.GHPoint{Lat: 1, Lon: 2.5}, res.GetSnappedPoint().GHPoint)

	// node number stays
	assertEqual(t, 3, queryGraph2.GetNodes())

	// snap directly to pillar node
	iter = expl.SetBaseNode(1)
	iter.Next()
	res = createLocationResult(2, 1.5, iter, 1, index.Pillar)
	queryGraph3 := lookup(g, res)
	assertGHPointEqual(t, util.GHPoint{Lat: 1.5, Lon: 1.5}, res.GetSnappedPoint().GHPoint)
	assertEqual(t, 3, res.GetClosestNode())
	assertEqual(t, 3, getPoints(queryGraph3, 0, 3).Size())
	assertEqual(t, 2, getPoints(queryGraph3, 3, 1).Size())

	res = createLocationResult(2, 1.7, iter, 1, index.Pillar)
	queryGraph4 := lookup(g, res)
	assertGHPointEqual(t, util.GHPoint{Lat: 1.5, Lon: 1.5}, res.GetSnappedPoint().GHPoint)
	assertEqual(t, 3, res.GetClosestNode())
	assertEqual(t, 3, getPoints(queryGraph4, 0, 3).Size())
	assertEqual(t, 2, getPoints(queryGraph4, 3, 1).Size())

	// snap to edge which has pillar nodes
	res = createLocationResult(1.5, 2, iter, 0, index.Edge)
	queryGraph5 := lookup(g, res)
	assertSnappedPointClose(t, 1.300019, 1.899962, res.GetSnappedPoint())
	assertEqual(t, 3, res.GetClosestNode())
	assertEqual(t, 4, getPoints(queryGraph5, 0, 3).Size())
	assertEqual(t, 2, getPoints(queryGraph5, 3, 1).Size())

	// snap to edge which has no pillar nodes
	iter = expl.SetBaseNode(2)
	iter.Next()
	res = createLocationResult(0.5, 0.1, iter, 0, index.Edge)
	queryGraph6 := lookup(g, res)
	assertSnappedPointClose(t, 0.5, 0, res.GetSnappedPoint())
	assertEqual(t, 3, res.GetClosestNode())
	assertEqual(t, 2, getPoints(queryGraph6, 0, 3).Size())
	assertEqual(t, 2, getPoints(queryGraph6, 3, 2).Size())
}

func TestFillVirtualEdges(t *testing.T) {
	g := setupTest(t)
	//       x (4)
	//  /*-*\
	// 0     1
	// |    /
	// 2  3
	na := g.GetNodeAccess()
	na.SetNode(0, 1, 0, math.NaN())
	na.SetNode(1, 1, 2.5, math.NaN())
	na.SetNode(2, 0, 0, math.NaN())
	na.SetNode(3, 0, 1, math.NaN())
	g.Edge(0, 2).SetDistance(10).SetDecimal(speedEnc, 60).SetReverseDecimal(speedEnc, 60)
	g.Edge(0, 1).SetDistance(10).SetDecimal(speedEnc, 60).SetReverseDecimal(speedEnc, 60).
		SetWayGeometry(util.CreatePointList(1.5, 1, 1.5, 1.5))
	g.Edge(1, 3)

	baseNode := 1
	iter := g.CreateEdgeExplorer(routingutil.AllEdges).SetBaseNode(baseNode)
	iter.Next()
	snap := createLocationResult(2, 1.7, iter, 1, index.Pillar)
	queryOverlay := BuildQueryOverlay(g, []*index.Snap{snap})
	realNodeMods := queryOverlay.getEdgeChangesAtRealNodes()
	assertEqual(t, 2, len(realNodeMods))

	queryGraph := Create(g, snap)
	state := getEdge(queryGraph, 0, 1)
	assertEqual(t, 4, state.FetchWayGeometry(util.FetchModeAll).Size())

	state = getEdge(queryGraph, 4, 3)
	assertEqual(t, 2, state.FetchWayGeometry(util.FetchModeAll).Size())
}

func TestMultipleVirtualNodes(t *testing.T) {
	g := setupTest(t)
	initGraph(g)

	iter := g.CreateEdgeExplorer(routingutil.AllEdges).SetBaseNode(1)
	iter.Next()
	res1 := createLocationResult(2, 1.7, iter, 1, index.Pillar)
	queryGraph := lookup(g, res1)
	assertGHPointEqual(t, util.GHPoint{Lat: 1.5, Lon: 1.5}, res1.GetSnappedPoint().GHPoint)
	assertEqual(t, 3, res1.GetClosestNode())
	assertEqual(t, 3, getPoints(queryGraph, 0, 3).Size())
	pl := getPoints(queryGraph, 3, 1)
	assertEqual(t, 2, pl.Size())
	assertGHPointEqual(t, util.GHPoint{Lat: 1.5, Lon: 1.5}, util.GHPoint{Lat: pl.GetLat(0), Lon: pl.GetLon(0)})
	assertGHPointEqual(t, util.GHPoint{Lat: 1, Lon: 2.5}, util.GHPoint{Lat: pl.GetLat(1), Lon: pl.GetLon(1)})

	edge := getEdge(queryGraph, 3, 1)
	assertNotNil(t, queryGraph.GetEdgeIteratorState(edge.GetEdge(), 3))
	assertNotNil(t, queryGraph.GetEdgeIteratorState(edge.GetEdge(), 1))

	edge = getEdge(queryGraph, 3, 0)
	assertNotNil(t, queryGraph.GetEdgeIteratorState(edge.GetEdge(), 3))
	assertNotNil(t, queryGraph.GetEdgeIteratorState(edge.GetEdge(), 0))

	// snap again => new virtual node on same edge!
	iter = g.CreateEdgeExplorer(routingutil.AllEdges).SetBaseNode(1)
	iter.Next()
	res1 = createLocationResult(2, 1.7, iter, 1, index.Pillar)
	res2 := createLocationResult(1.5, 2, iter, 0, index.Edge)
	queryGraph = lookup(g, res1, res2)
	assertEqual(t, 4, res2.GetClosestNode())
	assertSnappedPointClose(t, 1.300019, 1.899962, res2.GetSnappedPoint())
	assertEqual(t, 3, res1.GetClosestNode())
	assertGHPointEqual(t, util.GHPoint{Lat: 1.5, Lon: 1.5}, res1.GetSnappedPoint().GHPoint)

	assertEqual(t, 3, getPoints(queryGraph, 3, 0).Size())
	assertEqual(t, 2, getPoints(queryGraph, 3, 4).Size())
	assertEqual(t, 2, getPoints(queryGraph, 4, 1).Size())
	assertNil(t, getEdge(queryGraph, 4, 0))
	assertNil(t, getEdge(queryGraph, 3, 1))
}

func TestOneWay(t *testing.T) {
	g := setupTest(t)
	na := g.GetNodeAccess()
	na.SetNode(0, 0, 0, math.NaN())
	na.SetNode(1, 0, 1, math.NaN())
	g.Edge(0, 1).SetDistance(10).SetDecimal(speedEnc, 60)

	edge := getEdge(g, 0, 1)
	res1 := createLocationResult(0.1, 0.1, edge, 0, index.Edge)
	res2 := createLocationResult(0.1, 0.9, edge, 0, index.Edge)
	queryGraph := lookup(g, res2, res1)
	assertEqual(t, 2, res1.GetClosestNode())
	assertSnappedPointClose(t, 0, 0.1, res1.GetSnappedPoint())
	assertEqual(t, 3, res2.GetClosestNode())
	assertSnappedPointClose(t, 0, 0.9, res2.GetSnappedPoint())

	assertEqual(t, 2, getPoints(queryGraph, 0, 2).Size())
	assertEqual(t, 2, getPoints(queryGraph, 2, 3).Size())
	assertEqual(t, 2, getPoints(queryGraph, 3, 1).Size())
	assertNil(t, getEdge(queryGraph, 3, 0))
	assertNil(t, getEdge(queryGraph, 2, 1))
}

func TestVirtEdges(t *testing.T) {
	g := setupTest(t)
	initGraph(g)

	iter := g.CreateEdgeExplorer(routingutil.AllEdges).SetBaseNode(0)
	iter.Next()
	vEdges := []util.EdgeIteratorState{iter.Detach(false)}
	vi := NewVirtualEdgeIterator(routingutil.AllEdges, vEdges)
	assertTrue(t, vi.Next())
}

func TestEdgesShareOneNode(t *testing.T) {
	g := setupTest(t)
	initGraph(g)

	iter := getEdge(g, 0, 2)
	res1 := createLocationResult(0.5, 0, iter, 0, index.Edge)
	iter = getEdge(g, 1, 0)
	res2 := createLocationResult(1.5, 2, iter, 0, index.Edge)
	queryGraph := lookup(g, res1, res2)
	assertSnappedPointClose(t, 0.5, 0, res1.GetSnappedPoint())
	assertSnappedPointClose(t, 1.300019, 1.899962, res2.GetSnappedPoint())
	assertNotNil(t, getEdge(queryGraph, 0, 4))
	assertNotNil(t, getEdge(queryGraph, 0, 3))
}

func TestAvoidDuplicateVirtualNodesIfIdentical(t *testing.T) {
	g := setupTest(t)
	initGraph(g)

	edgeState := getEdge(g, 0, 2)
	res1 := createLocationResult(0.5, 0, edgeState, 0, index.Edge)
	res2 := createLocationResult(0.5, 0, edgeState, 0, index.Edge)
	lookup(g, res1, res2)
	assertSnappedPointClose(t, 0.5, 0, res1.GetSnappedPoint())
	assertSnappedPointClose(t, 0.5, 0, res2.GetSnappedPoint())
	assertEqual(t, 3, res1.GetClosestNode())
	assertEqual(t, 3, res2.GetClosestNode())

	// force skip due to tower node snapping in phase 2
	edgeState = getEdge(g, 0, 1)
	res1 = createLocationResult(1, 0, edgeState, 0, index.Tower)
	edgeState = getEdge(g, 0, 2)
	res2 = createLocationResult(0.5, 0, edgeState, 0, index.Edge)
	queryGraph := lookup(g, res1, res2)
	assertEqual(t, g.GetNodes()+1, queryGraph.GetNodes())
	qIter := queryGraph.CreateEdgeExplorer(routingutil.AllEdges).SetBaseNode(0)
	assertSetEqual(t, util.AsSet(1, 3), util.GetNeighbors(qIter))
}

func TestGetEdgeProps(t *testing.T) {
	g := setupTest(t)
	initGraph(g)
	e1 := getEdge(g, 0, 2)
	res1 := createLocationResult(0.5, 0, e1, 0, index.Edge)
	queryGraph := lookup(g, res1)
	// get virtual edge
	e1 = getEdge(queryGraph, res1.GetClosestNode(), 0)
	e2 := queryGraph.GetEdgeIteratorState(e1.GetEdge(), math.MinInt32)
	assertEqual(t, e1.GetEdge(), e2.GetEdge())
}

func TestInternalAPIOriginalEdgeKey(t *testing.T) {
	g := setupTest(t)
	initGraph(g)

	explorer := g.CreateEdgeExplorer(routingutil.AllEdges)
	iter := explorer.SetBaseNode(1)
	assertTrue(t, iter.Next())
	res := createLocationResult(2, 1.5, iter, 1, index.Pillar)
	queryGraph := lookup(g, res)

	assertGHPointEqual(t, util.GHPoint{Lat: 1.5, Lon: 1.5}, res.GetSnappedPoint().GHPoint)
	assertEqual(t, 3, res.GetClosestNode())

	qGraphExplorer := queryGraph.CreateEdgeExplorer(routingutil.AllEdges)
	qIter := qGraphExplorer.SetBaseNode(3)
	assertTrue(t, qIter.Next())
	assertEqual(t, 2, qIter.GetEdge())
	assertEqual(t, 0, qIter.GetAdjNode())
	veis := queryGraph.GetEdgeIteratorState(qIter.GetEdge(), 0).(*VirtualEdgeIteratorState)
	assertEqual(t, 3, veis.GetOriginalEdgeKey())

	assertTrue(t, qIter.Next())
	assertEqual(t, 3, qIter.GetEdge())
	assertEqual(t, 1, qIter.GetAdjNode())
	veis = queryGraph.GetEdgeIteratorState(qIter.GetEdge(), 1).(*VirtualEdgeIteratorState)
	assertEqual(t, 2, veis.GetOriginalEdgeKey())
}

func TestVirtualEdgeIds(t *testing.T) {
	// virtual nodes:     2
	//                0 - x - 1
	// virtual edges:   1   2
	accessEnc := ev.NewSimpleBooleanEncodedValueDir("access", true)
	spEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	em := routingutil.Start().Add(accessEnc).Add(spEnc).Build()
	g := storage.NewBaseGraphBuilder(em.BytesForFlags).CreateGraph()
	defer g.Close()
	na := g.GetNodeAccess()
	na.SetNode(0, 50.00, 10.10, math.NaN())
	na.SetNode(1, 50.00, 10.20, math.NaN())
	dist := util.DistEarth.CalcDist(na.GetLat(0), na.GetLon(0), na.GetLat(1), na.GetLon(1))
	edge := g.Edge(0, 1).SetDistance(dist).SetDecimal(spEnc, 60).SetReverseDecimal(spEnc, 60)
	edge.SetDecimal(spEnc, 50)
	edge.SetReverseDecimal(spEnc, 100)

	snap := createLocationResult(50.00, 10.15, edge, 0, index.Edge)
	queryGraph := CreateFromSnaps(g, []*index.Snap{snap})
	assertEqual(t, 3, queryGraph.GetNodes())
	assertEqual(t, 3, queryGraph.GetEdges())
	assertEqual(t, 4, len(queryGraph.GetVirtualEdges()))

	edge_0x := queryGraph.GetEdgeIteratorState(1, 2)
	edge_x0 := queryGraph.GetEdgeIteratorState(1, 0)
	edge_x1 := queryGraph.GetEdgeIteratorState(2, 1)
	edge_1x := queryGraph.GetEdgeIteratorState(2, 2)

	assertNodes(t, edge_0x, 0, 2)
	assertNodes(t, edge_x0, 2, 0)
	assertNodes(t, edge_x1, 2, 1)
	assertNodes(t, edge_1x, 1, 2)

	assertEqual(t, 1, edge_0x.GetEdge())
	assertEqual(t, 1, edge_x0.GetEdge())
	assertEqual(t, 2, edge_x1.GetEdge())
	assertEqual(t, 2, edge_1x.GetEdge())

	// edge keys
	assertEqual(t, 2, edge_0x.GetEdgeKey())
	assertEqual(t, 3, edge_x0.GetEdgeKey())
	assertEqual(t, 4, edge_x1.GetEdgeKey())
	assertEqual(t, 5, edge_1x.GetEdgeKey())
	assertNodes(t, queryGraph.GetEdgeIteratorStateForKey(2), 0, 2)
	assertNodes(t, queryGraph.GetEdgeIteratorStateForKey(3), 2, 0)
	assertNodes(t, queryGraph.GetEdgeIteratorStateForKey(4), 2, 1)
	assertNodes(t, queryGraph.GetEdgeIteratorStateForKey(5), 1, 2)

	// internally each edge is represented by two edge states
	if queryGraph.GetVirtualEdges()[0] != edge_0x {
		t.Fatal("expected same virtual edge [0]")
	}
	if queryGraph.GetVirtualEdges()[1] != edge_x0 {
		t.Fatal("expected same virtual edge [1]")
	}
	if queryGraph.GetVirtualEdges()[2] != edge_x1 {
		t.Fatal("expected same virtual edge [2]")
	}
	if queryGraph.GetVirtualEdges()[3] != edge_1x {
		t.Fatal("expected same virtual edge [3]")
	}

	for _, e := range []util.EdgeIteratorState{edge_0x, edge_x1} {
		assertNearF(t, 50, e.GetDecimal(spEnc), 1e-6)
		assertNearF(t, 100, e.GetReverseDecimal(spEnc), 1e-6)
	}

	for _, e := range []util.EdgeIteratorState{edge_x0, edge_1x} {
		assertNearF(t, 100, e.GetDecimal(spEnc), 1e-6)
		assertNearF(t, 50, e.GetReverseDecimal(spEnc), 1e-6)
	}

	assertPanics(t, func() {
		queryGraph.GetEdgeIteratorState(3, 2)
	})
}

func TestVirtualEdgeIdsReverse(t *testing.T) {
	// virtual nodes:     2
	//                0 - x - 1
	// virtual edges:   1   2
	accessEnc := ev.NewSimpleBooleanEncodedValueDir("access", true)
	spEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	em := routingutil.Start().Add(accessEnc).Add(spEnc).Build()
	g := storage.NewBaseGraphBuilder(em.BytesForFlags).CreateGraph()
	defer g.Close()
	na := g.GetNodeAccess()
	na.SetNode(0, 50.00, 10.10, math.NaN())
	na.SetNode(1, 50.00, 10.20, math.NaN())
	dist := util.DistEarth.CalcDist(na.GetLat(0), na.GetLon(0), na.GetLat(1), na.GetLon(1))
	// this time we store the edge the other way
	edge := g.Edge(1, 0).SetDistance(dist).SetDecimal(spEnc, 60).SetReverseDecimal(spEnc, 60)
	edge.SetDecimal(spEnc, 100).SetReverseDecimal(spEnc, 50)

	snap := createLocationResult(50.00, 10.15, edge, 0, index.Edge)
	queryGraph := CreateFromSnaps(g, []*index.Snap{snap})
	assertEqual(t, 3, queryGraph.GetNodes())
	assertEqual(t, 3, queryGraph.GetEdges())
	assertEqual(t, 4, len(queryGraph.GetVirtualEdges()))

	edge_0x := queryGraph.GetEdgeIteratorState(1, 2)
	edge_x0 := queryGraph.GetEdgeIteratorState(1, 0)
	edge_x1 := queryGraph.GetEdgeIteratorState(2, 1)
	edge_1x := queryGraph.GetEdgeIteratorState(2, 2)

	assertNodes(t, edge_0x, 0, 2)
	assertNodes(t, edge_x0, 2, 0)
	assertNodes(t, edge_x1, 2, 1)
	assertNodes(t, edge_1x, 1, 2)

	assertEqual(t, 1, edge_0x.GetEdge())
	assertEqual(t, 1, edge_x0.GetEdge())
	assertEqual(t, 2, edge_x1.GetEdge())
	assertEqual(t, 2, edge_1x.GetEdge())

	assertEqual(t, 2, edge_0x.GetEdgeKey())
	assertEqual(t, 3, edge_x0.GetEdgeKey())
	assertEqual(t, 4, edge_x1.GetEdgeKey())
	assertEqual(t, 5, edge_1x.GetEdgeKey())
	assertNodes(t, queryGraph.GetEdgeIteratorStateForKey(2), 0, 2)
	assertNodes(t, queryGraph.GetEdgeIteratorStateForKey(3), 2, 0)
	assertNodes(t, queryGraph.GetEdgeIteratorStateForKey(4), 2, 1)
	assertNodes(t, queryGraph.GetEdgeIteratorStateForKey(5), 1, 2)

	if queryGraph.GetVirtualEdges()[0] != edge_0x {
		t.Fatal("expected same virtual edge [0]")
	}
	if queryGraph.GetVirtualEdges()[1] != edge_x0 {
		t.Fatal("expected same virtual edge [1]")
	}
	if queryGraph.GetVirtualEdges()[2] != edge_x1 {
		t.Fatal("expected same virtual edge [2]")
	}
	if queryGraph.GetVirtualEdges()[3] != edge_1x {
		t.Fatal("expected same virtual edge [3]")
	}

	for _, e := range []util.EdgeIteratorState{edge_0x, edge_x1} {
		assertNearF(t, 50, e.GetDecimal(spEnc), 1e-6)
		assertNearF(t, 100, e.GetReverseDecimal(spEnc), 1e-6)
	}

	for _, e := range []util.EdgeIteratorState{edge_x0, edge_1x} {
		assertNearF(t, 100, e.GetDecimal(spEnc), 1e-6)
		assertNearF(t, 50, e.GetReverseDecimal(spEnc), 1e-6)
	}

	assertPanics(t, func() {
		queryGraph.GetEdgeIteratorState(3, 2)
	})
}

func TestUnfavoredEdgeDirections(t *testing.T) {
	g := setupTest(t)
	na := g.GetNodeAccess()
	// 0 <-> x <-> 1
	//       2
	na.SetNode(0, 0, 0, math.NaN())
	na.SetNode(1, 0, 2, math.NaN())
	edge := g.Edge(0, 1).SetDistance(10).SetDecimal(speedEnc, 60).SetReverseDecimal(speedEnc, 60)

	snap := fakeEdgeSnap(edge, 0, 1, 0)
	queryGraph := Create(g, snap)
	queryGraph.UnfavorVirtualEdge(1)

	assertTrue(t, getEdge(queryGraph, 2, 0).GetBool(util.UnfavoredEdge))
	assertTrue(t, getEdge(queryGraph, 2, 0).GetReverseBool(util.UnfavoredEdge))
	assertTrue(t, getEdge(queryGraph, 0, 2).GetBool(util.UnfavoredEdge))
	assertTrue(t, getEdge(queryGraph, 0, 2).GetReverseBool(util.UnfavoredEdge))

	assertFalse(t, getEdge(queryGraph, 2, 1).GetBool(util.UnfavoredEdge))
	assertFalse(t, getEdge(queryGraph, 2, 1).GetReverseBool(util.UnfavoredEdge))
	assertFalse(t, getEdge(queryGraph, 1, 2).GetBool(util.UnfavoredEdge))
	assertFalse(t, getEdge(queryGraph, 1, 2).GetReverseBool(util.UnfavoredEdge))
}

func TestUnfavorVirtualEdgePair(t *testing.T) {
	g := setupTest(t)
	//   ____
	//  |    |
	//  |    |
	//  0    1
	na := g.GetNodeAccess()
	na.SetNode(0, 0, 0, math.NaN())
	na.SetNode(1, 0, 2, math.NaN())
	g.Edge(0, 1).SetDistance(10).SetDecimal(speedEnc, 60).SetReverseDecimal(speedEnc, 60).
		SetWayGeometry(util.CreatePointList(2, 0, 2, 2))
	edge := getEdge(g, 0, 1)

	snap := fakeEdgeSnap(edge, 1.5, 0, 0)
	queryGraph := lookup(g, snap)

	queryGraph.UnfavorVirtualEdge(1)
	incomingEdge := queryGraph.GetEdgeIteratorState(1, 2).(*VirtualEdgeIteratorState)
	incomingEdgeReverse := queryGraph.GetEdgeIteratorState(1, incomingEdge.GetBaseNode()).(*VirtualEdgeIteratorState)
	assertTrue(t, incomingEdge.GetBool(util.UnfavoredEdge))
	assertTrue(t, incomingEdgeReverse.GetBool(util.UnfavoredEdge))

	unfavored := queryGraph.GetUnfavoredVirtualEdges()
	assertTrue(t, unfavored[incomingEdge])
	assertTrue(t, unfavored[incomingEdgeReverse])

	queryGraph.ClearUnfavoredStatus()
	assertFalse(t, incomingEdge.GetBool(util.UnfavoredEdge))
	assertFalse(t, incomingEdgeReverse.GetBool(util.UnfavoredEdge))
	assertEqual(t, 0, len(queryGraph.GetUnfavoredVirtualEdges()))
}

func TestIterationIssue163(t *testing.T) {
	g := setupTest(t)
	inEdgeFilter := routingutil.EdgeFilter(func(edge util.EdgeIteratorState) bool {
		return edge.GetReverseDecimal(speedEnc) > 0
	})
	outEdgeFilter := routingutil.EdgeFilter(func(edge util.EdgeIteratorState) bool {
		return edge.GetDecimal(speedEnc) > 0
	})
	inExplorer := g.CreateEdgeExplorer(inEdgeFilter)
	outExplorer := g.CreateEdgeExplorer(outEdgeFilter)

	nodeA := 0
	nodeB := 1

	g.GetNodeAccess().SetNode(nodeA, 1, 0, math.NaN())
	g.GetNodeAccess().SetNode(nodeB, 1, 10, math.NaN())
	g.Edge(nodeA, nodeB).SetDistance(10).SetDecimal(speedEnc, 60).
		SetWayGeometry(util.CreatePointList(1.5, 3, 1.5, 7))

	assertEdgeIdsStayingEqual(t, inExplorer, outExplorer, nodeA, nodeB)

	it := getEdge(g, nodeA, nodeB)
	snap1 := createLocationResult(1.5, 3, it, 1, index.Pillar)
	snap2 := createLocationResult(1.5, 7, it, 2, index.Pillar)

	q := lookup(g, snap1, snap2)
	nodeC := snap1.GetClosestNode()
	nodeD := snap2.GetClosestNode()

	inExplorer = q.CreateEdgeExplorer(inEdgeFilter)
	outExplorer = q.CreateEdgeExplorer(outEdgeFilter)

	assertEdgeIdsStayingEqual(t, inExplorer, outExplorer, nodeA, nodeC)
	assertEdgeIdsStayingEqual(t, inExplorer, outExplorer, nodeC, nodeD)
	assertEdgeIdsStayingEqual(t, inExplorer, outExplorer, nodeD, nodeB)
}

func TestTurnCostsProperlyPropagatedIssue282(t *testing.T) {
	spEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	turnCostEnc := ev.TurnCostCreate("car", 15)
	em := routingutil.Start().Add(spEnc).AddTurnCostEncodedValue(turnCostEnc).Build()
	graphWithTurnCosts := storage.NewBaseGraphBuilder(em.BytesForFlags).
		SetWithTurnCosts(true).CreateGraph()
	defer graphWithTurnCosts.Close()
	turnExt := graphWithTurnCosts.GetTurnCostStorage()
	na := graphWithTurnCosts.GetNodeAccess()
	na.SetNode(0, .00, .00, math.NaN())
	na.SetNode(1, .00, .01, math.NaN())
	na.SetNode(2, .01, .01, math.NaN())

	edge0 := graphWithTurnCosts.Edge(0, 1).SetDistance(10).SetDecimal(spEnc, 60).SetReverseDecimal(spEnc, 60)
	edge1 := graphWithTurnCosts.Edge(2, 1).SetDistance(10).SetDecimal(spEnc, 60).SetReverseDecimal(spEnc, 60)

	w := weighting.NewSpeedWeightingWithTurnCosts(spEnc, turnCostEnc, graphWithTurnCosts.GetTurnCostStorage(), na, math.Inf(1))

	assertNearF(t, 0, w.CalcTurnWeight(edge0.GetEdge(), 1, edge1.GetEdge()), 0.1)

	turnExt.SetDecimal(na, turnCostEnc, edge0.GetEdge(), 1, edge1.GetEdge(), 10)
	assertNearF(t, 10, w.CalcTurnWeight(edge0.GetEdge(), 1, edge1.GetEdge()), 0.1)

	res1 := createLocationResult(0.000, 0.005, edge0, 0, index.Edge)
	res2 := createLocationResult(0.005, 0.010, edge1, 0, index.Edge)
	qGraph := CreateFromSnaps(graphWithTurnCosts, []*index.Snap{res1, res2})
	wrappedWeighting := weighting.NewQueryGraphWeighting(graphWithTurnCosts, w, qGraph.GetClosestEdges())

	fromQueryEdge := getEdge(qGraph, res1.GetClosestNode(), 1).GetEdge()
	toQueryEdge := getEdge(qGraph, res2.GetClosestNode(), 1).GetEdge()

	assertNearF(t, 10, wrappedWeighting.CalcTurnWeight(fromQueryEdge, 1, toQueryEdge), 0.1)
}

func TestLoopStreetIssue151(t *testing.T) {
	g := setupTest(t)
	// 0--1--3--4
	//    |  |
	//    x---
	g.Edge(0, 1).SetDistance(10).SetDecimal(speedEnc, 60).SetReverseDecimal(speedEnc, 60)
	g.Edge(1, 3).SetDistance(10).SetDecimal(speedEnc, 60).SetReverseDecimal(speedEnc, 60)
	g.Edge(3, 4).SetDistance(10).SetDecimal(speedEnc, 60).SetReverseDecimal(speedEnc, 60)
	edge := g.Edge(1, 3).SetDistance(20).SetDecimal(speedEnc, 60).SetReverseDecimal(speedEnc, 60).
		SetWayGeometry(util.CreatePointList(-0.001, 0.001, -0.001, 0.002))
	updateDistancesFor(g, 0, 0, 0)
	updateDistancesFor(g, 1, 0, 0.001)
	updateDistancesFor(g, 3, 0, 0.002)
	updateDistancesFor(g, 4, 0, 0.003)

	snap := index.NewSnap(-0.0005, 0.001)
	snap.SetClosestEdge(edge)
	snap.SetWayIndex(1)
	snap.CalcSnappedPoint(util.DistEarth)

	qg := lookup(g, snap)
	ee := qg.CreateEdgeExplorer(routingutil.AllEdges)
	assertSetEqual(t, util.AsSet(0, 5, 3), util.GetNeighbors(ee.SetBaseNode(1)))
}

func TestVirtualEdgeDistance(t *testing.T) {
	g := setupTest(t)
	//   x
	// -----
	// |   |
	// 0   1
	na := g.GetNodeAccess()
	na.SetNode(0, 0, 0, math.NaN())
	na.SetNode(1, 0, 1, math.NaN())
	// dummy node for valid graph bounds
	na.SetNode(2, 2, 2, math.NaN())
	dist := 0.0
	dist += util.DistPlane.CalcDist(0, 0, 1, 0)
	dist += util.DistPlane.CalcDist(1, 0, 1, 1)
	dist += util.DistPlane.CalcDist(1, 1, 0, 1)
	g.Edge(0, 1).SetDistance(dist).SetDecimal(speedEnc, 60).SetReverseDecimal(speedEnc, 60).
		SetWayGeometry(util.CreatePointList(1, 0, 1, 1))
	locIndex := index.NewLocationIndexTree(g, storage.NewRAMDirectory("", false))
	locIndex.PrepareIndex()
	snap := locIndex.FindClosest(1.01, 0.7, routingutil.AllEdges)
	queryGraph := lookup(g, snap)
	iter := queryGraph.CreateEdgeExplorer(routingutil.AllEdges).SetBaseNode(3)
	virtualEdgeDistanceSum := 0.0
	for iter.Next() {
		virtualEdgeDistanceSum += iter.GetDistance()
	}
	directDist := g.GetEdgeIteratorState(0, 1).GetDistance()
	assertNearF(t, directDist, virtualEdgeDistanceSum, 1e-3)
}

// --- helpers ---

func fakeEdgeSnap(edge util.EdgeIteratorState, lat, lon float64, wayIndex int) *index.Snap {
	snap := index.NewSnap(lat, lon)
	snap.SetClosestEdge(edge)
	snap.SetWayIndex(wayIndex)
	snap.SetSnappedPosition(index.Edge)
	snap.CalcSnappedPoint(util.DistEarth)
	return snap
}

func assertEdgeIdsStayingEqual(t *testing.T, inExplorer, outExplorer util.EdgeExplorer, startNode, endNode int) {
	t.Helper()
	it := outExplorer.SetBaseNode(startNode)
	it.Next()
	assertEqual(t, startNode, it.GetBaseNode())
	assertEqual(t, endNode, it.GetAdjNode())
	expectedEdgeID := it.GetEdge()
	assertFalse(t, it.Next())

	it = inExplorer.SetBaseNode(endNode)
	it.Next()
	assertEqual(t, endNode, it.GetBaseNode())
	assertEqual(t, startNode, it.GetAdjNode())
	if expectedEdgeID != it.GetEdge() {
		t.Fatalf("the edge id is not the same: want %d, got %d", expectedEdgeID, it.GetEdge())
	}
	assertFalse(t, it.Next())
}

func updateDistancesFor(g *storage.BaseGraph, node int, lat, lon float64) {
	na := g.GetNodeAccess()
	na.SetNode(node, lat, lon, math.NaN())
	iter := g.CreateEdgeExplorer(routingutil.AllEdges).SetBaseNode(node)
	for iter.Next() {
		pl := iter.FetchWayGeometry(util.FetchModeAll)
		iter.SetDistance(util.DistEarth.CalcPointListDistance(pl))
	}
}

func assertNodes(t *testing.T, edge util.EdgeIteratorState, base, adj int) {
	t.Helper()
	assertEqual(t, base, edge.GetBaseNode())
	assertEqual(t, adj, edge.GetAdjNode())
}

func assertEqual(t *testing.T, expected, actual int) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %d, got %d", expected, actual)
	}
}

func assertGHPointEqual(t *testing.T, expected, actual util.GHPoint) {
	t.Helper()
	if math.Abs(expected.Lat-actual.Lat) > 1e-6 || math.Abs(expected.Lon-actual.Lon) > 1e-6 {
		t.Fatalf("expected GHPoint(%v, %v), got (%v, %v)", expected.Lat, expected.Lon, actual.Lat, actual.Lon)
	}
}

func assertSnappedPointClose(t *testing.T, lat, lon float64, actual util.GHPoint3D) {
	t.Helper()
	if math.Abs(lat-actual.Lat) > 1e-4 || math.Abs(lon-actual.Lon) > 1e-4 {
		t.Fatalf("expected snapped point near (%v, %v), got (%v, %v)", lat, lon, actual.Lat, actual.Lon)
	}
}

func assertNearF(t *testing.T, expected, actual, delta float64) {
	t.Helper()
	if math.Abs(expected-actual) > delta {
		t.Fatalf("expected %v +/- %v, got %v", expected, actual, delta)
	}
}

func assertTrue(t *testing.T, v bool) {
	t.Helper()
	if !v {
		t.Fatal("expected true")
	}
}

func assertFalse(t *testing.T, v bool) {
	t.Helper()
	if v {
		t.Fatal("expected false")
	}
}

func assertNotNil(t *testing.T, v any) {
	t.Helper()
	if v == nil {
		t.Fatal("expected non-nil")
	}
}

func assertNil(t *testing.T, v any) {
	t.Helper()
	if v != nil {
		t.Fatal("expected nil")
	}
}

func assertPanics(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	f()
}

func assertSetEqual(t *testing.T, expected, actual map[int]bool) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Fatalf("expected set %v, got %v", expected, actual)
	}
	for k := range expected {
		if !actual[k] {
			t.Fatalf("expected set %v, got %v", expected, actual)
		}
	}
}
