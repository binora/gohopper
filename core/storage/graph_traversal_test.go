package storage

import (
	"math"
	"testing"

	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/util"
)

// --- Test infrastructure ---

type testGraph struct {
	graph          *BaseGraph
	carAccessEnc   ev.BooleanEncodedValue
	carSpeedEnc    ev.DecimalEncodedValue
	footAccessEnc  ev.BooleanEncodedValue
	footSpeedEnc   ev.DecimalEncodedValue
	carOutExplorer util.EdgeExplorer
	carInExplorer  util.EdgeExplorer
	carAllExplorer util.EdgeExplorer
}

func newTestGraph(t *testing.T) *testGraph {
	t.Helper()
	return newTestGraph3D(t, false)
}

func newTestGraph3D(t *testing.T, is3D bool) *testGraph {
	t.Helper()
	carAccess := ev.NewSimpleBooleanEncodedValueDir("car_access", true)
	carSpeed := ev.NewDecimalEncodedValueImpl("car_speed", 5, 5, false)
	footAccess := ev.NewSimpleBooleanEncodedValueDir("foot_access", true)
	footSpeed := ev.NewDecimalEncodedValueImpl("foot_speed", 4, 1, true)

	cfg := ev.NewInitializerConfig()
	carAccess.Init(cfg)
	carSpeed.Init(cfg)
	footAccess.Init(cfg)
	footSpeed.Init(cfg)

	bytesForFlags := cfg.GetRequiredBytes()
	g := NewBaseGraphBuilder(bytesForFlags).SetWithElevation(is3D).CreateGraph()
	t.Cleanup(func() { g.Close() })

	carOutFilter := accessOutFilter(carAccess)
	carInFilter := accessInFilter(carAccess)

	tg := &testGraph{
		graph:          g,
		carAccessEnc:   carAccess,
		carSpeedEnc:    carSpeed,
		footAccessEnc:  footAccess,
		footSpeedEnc:   footSpeed,
		carOutExplorer: g.CreateEdgeExplorer(carOutFilter),
		carInExplorer:  g.CreateEdgeExplorer(carInFilter),
		carAllExplorer: g.CreateEdgeExplorer(routingutil.AllEdges),
	}
	return tg
}

func accessOutFilter(accessEnc ev.BooleanEncodedValue) routingutil.EdgeFilter {
	return func(edge util.EdgeIteratorState) bool {
		return edge.GetBool(accessEnc)
	}
}

func accessInFilter(accessEnc ev.BooleanEncodedValue) routingutil.EdgeFilter {
	return func(edge util.EdgeIteratorState) bool {
		return edge.GetReverseBool(accessEnc)
	}
}

func (tg *testGraph) countOut(node int) int {
	return util.Count(tg.carOutExplorer.SetBaseNode(node))
}

func (tg *testGraph) countIn(node int) int {
	return util.Count(tg.carInExplorer.SetBaseNode(node))
}

func (tg *testGraph) countAll(node int) int {
	return util.Count(tg.carAllExplorer.SetBaseNode(node))
}

// getEdge finds an edge between base and adj nodes.
func getEdge(g *BaseGraph, base, adj int) util.EdgeIteratorState {
	explorer := g.CreateEdgeExplorer(routingutil.AllEdges)
	count := util.CountAdj(explorer.SetBaseNode(base), adj)
	if count > 1 {
		panic("multiple edges between nodes")
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
	panic("should not reach here")
}

func assertPList(t *testing.T, expected, actual *util.PointList) {
	t.Helper()
	if expected.Size() != actual.Size() {
		t.Fatalf("pointlist size mismatch: expected %d, got %d\n  expected: %s\n  actual: %s",
			expected.Size(), actual.Size(), expected, actual)
	}
	for i := range expected.Size() {
		if math.Abs(expected.GetLat(i)-actual.GetLat(i)) > 1e-5 {
			t.Fatalf("lat[%d] mismatch: expected %f, got %f", i, expected.GetLat(i), actual.GetLat(i))
		}
		if math.Abs(expected.GetLon(i)-actual.GetLon(i)) > 1e-5 {
			t.Fatalf("lon[%d] mismatch: expected %f, got %f", i, expected.GetLon(i), actual.GetLon(i))
		}
	}
}

func assertPList3D(t *testing.T, expected, actual *util.PointList) {
	t.Helper()
	assertPList(t, expected, actual)
	for i := range expected.Size() {
		if math.Abs(expected.GetEle(i)-actual.GetEle(i)) > 0.1 {
			t.Fatalf("ele[%d] mismatch: expected %f, got %f", i, expected.GetEle(i), actual.GetEle(i))
		}
	}
}

func setsEqual(a, b map[int]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

func assertEqual(t *testing.T, expected, actual int) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %d, got %d", expected, actual)
	}
}

func assertEqualF(t *testing.T, expected, actual, delta float64) {
	t.Helper()
	if math.Abs(expected-actual) > delta {
		t.Fatalf("expected %f (±%f), got %f", expected, delta, actual)
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

func assertPanics(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	f()
}

// --- Wave 1: Basic tests ---

func TestSetTooBigDistance(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph

	maxDist := MaxDist
	edge1 := g.Edge(0, 1).SetDistance(maxDist)
	assertEqualF(t, maxDist, edge1.GetDistance(), 1)

	edge2 := g.Edge(0, 2).SetDistance(maxDist + 1)
	assertEqualF(t, maxDist, edge2.GetDistance(), 1)
}

func TestSetNodes(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	na := g.GetNodeAccess()
	for i := range tg.graph.Store.nodeCount + 200 {
		na.SetNode(i, float64(2*i), float64(3*i), 0)
	}
	n := 100 + 1
	g.Edge(n, n+1).SetDistance(10)
	g.Edge(n, n+2).SetDistance(10)
	assertEqual(t, 2, tg.countAll(n))
}

func TestCreateLocation(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	g.Edge(3, 1).SetDistance(50).SetBoolBothDir(tg.carAccessEnc, true, true)
	assertEqual(t, 1, tg.countOut(1))

	g.Edge(1, 2).SetDistance(100).SetBoolBothDir(tg.carAccessEnc, true, true)
	assertEqual(t, 2, tg.countOut(1))
}

func TestEdges(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	g.Edge(2, 1).SetDistance(12).SetBoolBothDir(tg.carAccessEnc, true, true)
	assertEqual(t, 1, tg.countOut(2))

	g.Edge(2, 3).SetDistance(12).SetBoolBothDir(tg.carAccessEnc, true, true)
	assertEqual(t, 1, tg.countOut(1))
	assertEqual(t, 2, tg.countOut(2))
	assertEqual(t, 1, tg.countOut(3))
}

func TestUnidirectional(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph

	g.Edge(1, 2).SetDistance(12).SetBoolBothDir(tg.carAccessEnc, true, false)
	g.Edge(1, 11).SetDistance(12).SetBoolBothDir(tg.carAccessEnc, true, false)
	g.Edge(11, 1).SetDistance(12).SetBoolBothDir(tg.carAccessEnc, true, false)
	g.Edge(1, 12).SetDistance(12).SetBoolBothDir(tg.carAccessEnc, true, false)
	g.Edge(3, 2).SetDistance(112).SetBoolBothDir(tg.carAccessEnc, true, false)

	i := tg.carOutExplorer.SetBaseNode(2)
	assertFalse(t, i.Next())

	assertEqual(t, 1, tg.countIn(1))
	assertEqual(t, 2, tg.countIn(2))
	assertEqual(t, 0, tg.countIn(3))

	assertEqual(t, 3, tg.countOut(1))
	assertEqual(t, 0, tg.countOut(2))
	assertEqual(t, 1, tg.countOut(3))

	i = tg.carOutExplorer.SetBaseNode(3)
	assertTrue(t, i.Next())
	assertEqual(t, 2, i.GetAdjNode())

	i = tg.carOutExplorer.SetBaseNode(1)
	assertTrue(t, i.Next())
	assertEqual(t, 12, i.GetAdjNode())
	assertTrue(t, i.Next())
	assertEqual(t, 11, i.GetAdjNode())
	assertTrue(t, i.Next())
	assertEqual(t, 2, i.GetAdjNode())
	assertFalse(t, i.Next())
}

func TestDirectional(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	g.Edge(1, 2).SetDistance(12).SetBoolBothDir(tg.carAccessEnc, true, true)
	g.Edge(2, 3).SetDistance(12).SetBoolBothDir(tg.carAccessEnc, true, false)
	g.Edge(3, 4).SetDistance(12).SetBoolBothDir(tg.carAccessEnc, true, false)
	g.Edge(3, 5).SetDistance(12).SetBoolBothDir(tg.carAccessEnc, true, true)
	g.Edge(6, 3).SetDistance(12).SetBoolBothDir(tg.carAccessEnc, true, false)

	assertEqual(t, 1, tg.countAll(1))
	assertEqual(t, 1, tg.countIn(1))
	assertEqual(t, 1, tg.countOut(1))

	assertEqual(t, 2, tg.countAll(2))
	assertEqual(t, 1, tg.countIn(2))
	assertEqual(t, 2, tg.countOut(2))

	assertEqual(t, 4, tg.countAll(3))
	assertEqual(t, 3, tg.countIn(3))
	assertEqual(t, 2, tg.countOut(3))

	assertEqual(t, 1, tg.countAll(4))
	assertEqual(t, 1, tg.countIn(4))
	assertEqual(t, 0, tg.countOut(4))

	assertEqual(t, 1, tg.countAll(5))
	assertEqual(t, 1, tg.countIn(5))
	assertEqual(t, 1, tg.countOut(5))
}

func TestDozendEdges(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	g.Edge(1, 2).SetDistance(12).SetBoolBothDir(tg.carAccessEnc, true, true)
	assertEqual(t, 1, tg.countAll(1))

	g.Edge(1, 3).SetDistance(13).SetBoolBothDir(tg.carAccessEnc, true, false)
	assertEqual(t, 2, tg.countAll(1))
	g.Edge(1, 4).SetDistance(14).SetBoolBothDir(tg.carAccessEnc, true, false)
	assertEqual(t, 3, tg.countAll(1))
	g.Edge(1, 5).SetDistance(15).SetBoolBothDir(tg.carAccessEnc, true, false)
	assertEqual(t, 4, tg.countAll(1))
	g.Edge(1, 6).SetDistance(16).SetBoolBothDir(tg.carAccessEnc, true, false)
	assertEqual(t, 5, tg.countAll(1))
	g.Edge(1, 7).SetDistance(16).SetBoolBothDir(tg.carAccessEnc, true, false)
	assertEqual(t, 6, tg.countAll(1))
	g.Edge(1, 8).SetDistance(16).SetBoolBothDir(tg.carAccessEnc, true, false)
	assertEqual(t, 7, tg.countAll(1))
	g.Edge(1, 9).SetDistance(16).SetBoolBothDir(tg.carAccessEnc, true, false)
	assertEqual(t, 8, tg.countAll(1))
	assertEqual(t, 8, tg.countOut(1))
	assertEqual(t, 1, tg.countIn(1))
	assertEqual(t, 1, tg.countIn(2))
}

func TestCheckFirstNode(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	assertPanics(t, func() { tg.countAll(1) })
	g.GetNodeAccess().SetNode(1, 0, 0, 0)
	assertEqual(t, 0, tg.countAll(1))
	g.Edge(0, 1).SetDistance(12).SetBoolBothDir(tg.carAccessEnc, true, true)
	assertEqual(t, 1, tg.countAll(1))
}

func TestBoundsTraversal(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	b := g.GetBounds()
	assertEqualF(t, util.CreateInverse(false).MaxLat, b.MaxLat, 1e-6)

	na := g.GetNodeAccess()
	na.SetNode(0, 10, 20, 0)
	b = g.GetBounds()
	assertEqualF(t, 10, b.MaxLat, 1e-6)
	assertEqualF(t, 20, b.MaxLon, 1e-6)

	na.SetNode(0, 15, -15, 0)
	b = g.GetBounds()
	assertEqualF(t, 15, b.MaxLat, 1e-6)
	assertEqualF(t, 20, b.MaxLon, 1e-6)
	assertEqualF(t, 10, b.MinLat, 1e-6)
	assertEqualF(t, -15, b.MinLon, 1e-6)
}

func TestEdgeProperties(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	iter1 := g.Edge(0, 1).SetDistance(10).SetBoolBothDir(tg.carAccessEnc, true, true)
	iter2 := g.Edge(0, 2).SetDistance(20).SetBoolBothDir(tg.carAccessEnc, true, true)

	edgeID := iter1.GetEdge()
	iter := g.GetEdgeIteratorState(edgeID, 0)
	assertEqualF(t, 10, iter.GetDistance(), 1e-5)

	edgeID = iter2.GetEdge()
	iter = g.GetEdgeIteratorState(edgeID, 0)
	assertEqual(t, 2, iter.GetBaseNode())
	assertEqual(t, 0, iter.GetAdjNode())
	assertEqualF(t, 20, iter.GetDistance(), 1e-5)

	iter = g.GetEdgeIteratorState(edgeID, 2)
	assertEqual(t, 0, iter.GetBaseNode())
	assertEqual(t, 2, iter.GetAdjNode())
	assertEqualF(t, 20, iter.GetDistance(), 1e-5)

	iter = g.GetEdgeIteratorState(edgeID, math.MinInt32)
	if iter == nil {
		t.Fatal("expected non-nil for MinInt32")
	}
	assertEqual(t, 0, iter.GetBaseNode())
	assertEqual(t, 2, iter.GetAdjNode())

	iter = g.GetEdgeIteratorState(edgeID, 1)
	if iter != nil {
		t.Fatal("expected nil for invalid adj node")
	}
}

func TestCreateDuplicateEdges(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	g.Edge(2, 1).SetDistance(12).SetBoolBothDir(tg.carAccessEnc, true, true)
	g.Edge(2, 3).SetDistance(12).SetBoolBothDir(tg.carAccessEnc, true, true)
	g.Edge(2, 3).SetDistance(13).SetBoolBothDir(tg.carAccessEnc, true, false)
	assertEqual(t, 3, tg.countOut(2))

	// no exception
	g.GetEdgeIteratorState(1, 3)

	// should panic for out of bounds
	assertPanics(t, func() { g.GetEdgeIteratorState(4, 3) })
	assertPanics(t, func() { g.GetEdgeIteratorState(-1, 3) })

	iter := tg.carOutExplorer.SetBaseNode(2)
	assertTrue(t, iter.Next())
	oneIter := g.GetEdgeIteratorState(iter.GetEdge(), 3)
	assertEqualF(t, 13, oneIter.GetDistance(), 1e-6)
	assertEqual(t, 2, oneIter.GetBaseNode())
	assertTrue(t, oneIter.GetBool(tg.carAccessEnc))
	assertFalse(t, oneIter.GetReverseBool(tg.carAccessEnc))

	oneIter = g.GetEdgeIteratorState(iter.GetEdge(), 2)
	assertEqualF(t, 13, oneIter.GetDistance(), 1e-6)
	assertEqual(t, 3, oneIter.GetBaseNode())
	assertFalse(t, oneIter.GetBool(tg.carAccessEnc))
	assertTrue(t, oneIter.GetReverseBool(tg.carAccessEnc))

	g.Edge(3, 2).SetDistance(14).SetBoolBothDir(tg.carAccessEnc, true, true)
	assertEqual(t, 4, tg.countOut(2))
}

// --- Wave 2: Iterator tests ---

func TestUnidirectionalEdgeFilter(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	g.Edge(1, 2).SetDistance(12).SetBoolBothDir(tg.carAccessEnc, true, false)
	g.Edge(1, 11).SetDistance(12).SetBoolBothDir(tg.carAccessEnc, true, false)
	g.Edge(11, 1).SetDistance(12).SetBoolBothDir(tg.carAccessEnc, true, false)
	g.Edge(1, 12).SetDistance(12).SetBoolBothDir(tg.carAccessEnc, true, false)
	g.Edge(3, 2).SetDistance(112).SetBoolBothDir(tg.carAccessEnc, true, false)

	i := tg.carOutExplorer.SetBaseNode(2)
	assertFalse(t, i.Next())
	assertEqual(t, 4, tg.countAll(1))
	assertEqual(t, 1, tg.countIn(1))
	assertEqual(t, 2, tg.countIn(2))
	assertEqual(t, 0, tg.countIn(3))
	assertEqual(t, 3, tg.countOut(1))
	assertEqual(t, 0, tg.countOut(2))
	assertEqual(t, 1, tg.countOut(3))
}

func TestUpdateUnidirectional(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	g.Edge(1, 2).SetDistance(12).SetBoolBothDir(tg.carAccessEnc, true, false)
	g.Edge(3, 2).SetDistance(112).SetBoolBothDir(tg.carAccessEnc, true, false)

	i := tg.carOutExplorer.SetBaseNode(2)
	assertFalse(t, i.Next())
	i = tg.carOutExplorer.SetBaseNode(3)
	assertTrue(t, i.Next())
	assertEqual(t, 2, i.GetAdjNode())
	assertFalse(t, i.Next())

	g.Edge(2, 3).SetDistance(112).SetBoolBothDir(tg.carAccessEnc, true, false)
	i = tg.carOutExplorer.SetBaseNode(2)
	assertTrue(t, i.Next())
	assertEqual(t, 3, i.GetAdjNode())
	i = tg.carOutExplorer.SetBaseNode(3)
	assertTrue(t, i.Next())
	assertEqual(t, 2, i.GetAdjNode())
	assertFalse(t, i.Next())
}

func TestCopyProperties(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	edge := g.Edge(1, 3).SetDistance(10).SetBoolBothDir(tg.carAccessEnc, true, false)
	edge.SetKeyValues(map[string]any{"name": "testing"})
	edge.SetWayGeometry(util.CreatePointList(1, 2))

	newEdge := g.Edge(1, 3).SetDistance(10).SetBoolBothDir(tg.carAccessEnc, true, false)
	newEdge.CopyPropertiesFrom(edge)

	if edge.GetName() != newEdge.GetName() {
		t.Fatalf("name mismatch: %s vs %s", edge.GetName(), newEdge.GetName())
	}
	assertEqualF(t, edge.GetDistance(), newEdge.GetDistance(), 1e-7)
	assertPList(t, edge.FetchWayGeometry(util.FetchModePillarOnly), newEdge.FetchWayGeometry(util.FetchModePillarOnly))
}

func TestGetLocations(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	na := g.GetNodeAccess()
	na.SetNode(0, 12, 23, 0)
	na.SetNode(1, 22, 23, 0)
	assertEqual(t, 2, g.GetNodes())

	g.Edge(0, 1).SetDistance(10).SetBoolBothDir(tg.carAccessEnc, true, true)
	assertEqual(t, 2, g.GetNodes())

	g.Edge(0, 2).SetDistance(10).SetBoolBothDir(tg.carAccessEnc, true, true)
	assertEqual(t, 3, g.GetNodes())
}

func TestAddLocation(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	initExampleGraph(tg)
	checkExampleGraph(t, tg, g)
}

func initExampleGraph(tg *testGraph) {
	g := tg.graph
	na := g.GetNodeAccess()
	na.SetNode(0, 12, 23, 0)
	na.SetNode(1, 38.33, 135.3, 0)
	na.SetNode(2, 6, 139, 0)
	na.SetNode(3, 78, 89, 0)
	na.SetNode(4, 2, 1, 0)
	na.SetNode(5, 7, 5, 0)
	g.Edge(0, 1).SetDistance(12).SetBoolBothDir(tg.carAccessEnc, true, true)
	g.Edge(0, 2).SetDistance(212).SetBoolBothDir(tg.carAccessEnc, true, true)
	g.Edge(0, 3).SetDistance(212).SetBoolBothDir(tg.carAccessEnc, true, true)
	g.Edge(0, 4).SetDistance(212).SetBoolBothDir(tg.carAccessEnc, true, true)
	g.Edge(0, 5).SetDistance(212).SetBoolBothDir(tg.carAccessEnc, true, true)
}

func checkExampleGraph(t *testing.T, tg *testGraph, g *BaseGraph) {
	t.Helper()
	na := g.GetNodeAccess()
	assertEqualF(t, 12, na.GetLat(0), 1e-6)
	assertEqualF(t, 23, na.GetLon(0), 1e-6)
	assertEqualF(t, 38.33, na.GetLat(1), 1e-3)
	assertEqualF(t, 135.3, na.GetLon(1), 1e-3)
	assertEqualF(t, 6, na.GetLat(2), 1e-6)
	assertEqualF(t, 139, na.GetLon(2), 1e-6)
	assertEqualF(t, 78, na.GetLat(3), 1e-6)
	assertEqualF(t, 89, na.GetLon(3), 1e-6)

	neighbors1 := util.GetNeighbors(tg.carOutExplorer.SetBaseNode(1))
	if !setsEqual(neighbors1, util.AsSet(0)) {
		t.Fatalf("expected neighbors {0} for node 1, got %v", neighbors1)
	}
	neighbors0 := util.GetNeighbors(tg.carOutExplorer.SetBaseNode(0))
	if !setsEqual(neighbors0, util.AsSet(5, 4, 3, 2, 1)) {
		t.Fatalf("expected neighbors {1,2,3,4,5} for node 0, got %v", neighbors0)
	}
}

func TestEdgeReturn(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	iter := g.Edge(4, 10).SetDistance(100)
	iter.SetBoolBothDir(tg.carAccessEnc, true, false)
	assertEqual(t, 4, iter.GetBaseNode())
	assertEqual(t, 10, iter.GetAdjNode())

	iter = g.Edge(14, 10).SetDistance(100)
	iter.SetBoolBothDir(tg.carAccessEnc, true, false)
	assertEqual(t, 14, iter.GetBaseNode())
	assertEqual(t, 10, iter.GetAdjNode())
}

func TestFlags(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph

	g.Edge(0, 1).SetBoolBothDir(tg.carAccessEnc, true, true).SetDistance(10).SetDecimal(tg.carSpeedEnc, 100)
	g.Edge(2, 3).SetBoolBothDir(tg.carAccessEnc, true, false).SetDistance(10).SetDecimal(tg.carSpeedEnc, 10)

	iter := tg.carAllExplorer.SetBaseNode(0)
	assertTrue(t, iter.Next())
	assertEqualF(t, 100, iter.GetDecimal(tg.carSpeedEnc), 1)
	assertTrue(t, iter.GetBool(tg.carAccessEnc))
	assertTrue(t, iter.GetReverseBool(tg.carAccessEnc))

	iter = tg.carAllExplorer.SetBaseNode(2)
	assertTrue(t, iter.Next())
	assertEqualF(t, 10, iter.GetDecimal(tg.carSpeedEnc), 1)
	assertTrue(t, iter.GetBool(tg.carAccessEnc))
	assertFalse(t, iter.GetReverseBool(tg.carAccessEnc))

	assertPanics(t, func() { g.Edge(0, 1).SetDistance(-1) })
}

func TestFootMix(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	g.Edge(0, 1).SetDistance(10).SetBoolBothDir(tg.footAccessEnc, true, true)
	g.Edge(0, 2).SetDistance(10).SetBoolBothDir(tg.carAccessEnc, true, true)
	edge := g.Edge(0, 3).SetDistance(10)
	edge.SetBoolBothDir(tg.footAccessEnc, true, true)
	edge.SetBoolBothDir(tg.carAccessEnc, true, true)

	footOutExplorer := g.CreateEdgeExplorer(accessOutFilter(tg.footAccessEnc))
	footNeighbors := util.GetNeighbors(footOutExplorer.SetBaseNode(0))
	if !setsEqual(footNeighbors, util.AsSet(3, 1)) {
		t.Fatalf("expected foot neighbors {1,3}, got %v", footNeighbors)
	}
	carNeighbors := util.GetNeighbors(tg.carOutExplorer.SetBaseNode(0))
	if !setsEqual(carNeighbors, util.AsSet(3, 2)) {
		t.Fatalf("expected car neighbors {2,3}, got %v", carNeighbors)
	}
}

func TestGetAllEdges(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	g.Edge(0, 1).SetDistance(2).SetBoolBothDir(tg.carAccessEnc, true, true)
	g.Edge(3, 1).SetDistance(1).SetBoolBothDir(tg.carAccessEnc, true, false)
	g.Edge(3, 2).SetDistance(1).SetBoolBothDir(tg.carAccessEnc, true, false)

	iter := g.GetAllEdges()
	assertTrue(t, iter.Next())
	edgeID := iter.GetEdge()
	assertEqual(t, 0, iter.GetBaseNode())
	assertEqual(t, 1, iter.GetAdjNode())
	assertEqualF(t, 2, iter.GetDistance(), 1e-6)

	assertTrue(t, iter.Next())
	edgeID2 := iter.GetEdge()
	assertEqual(t, 1, edgeID2-edgeID)
	assertEqual(t, 3, iter.GetBaseNode())
	assertEqual(t, 1, iter.GetAdjNode())

	assertTrue(t, iter.Next())
	assertEqual(t, 3, iter.GetBaseNode())
	assertEqual(t, 2, iter.GetAdjNode())

	assertFalse(t, iter.Next())
}

func TestKVStorage(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	iter1 := g.Edge(0, 1).SetDistance(10).SetBoolBothDir(tg.carAccessEnc, true, true)
	iter1.SetKeyValues(map[string]any{"name": "named street1"})

	iter2 := g.Edge(0, 1).SetDistance(10).SetBoolBothDir(tg.carAccessEnc, true, true)
	iter2.SetKeyValues(map[string]any{"name": "named street2"})

	e1 := g.GetEdgeIteratorState(iter1.GetEdge(), iter1.GetAdjNode())
	if e1.GetName() != "named street1" {
		t.Fatalf("expected 'named street1', got '%s'", e1.GetName())
	}
	e2 := g.GetEdgeIteratorState(iter2.GetEdge(), iter2.GetAdjNode())
	if e2.GetName() != "named street2" {
		t.Fatalf("expected 'named street2', got '%s'", e2.GetName())
	}
}

func TestPropertiesWithNoInit(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	edge := g.Edge(0, 1)
	impl := edge.(*EdgeIteratorStateImpl)
	flags := impl.GetFlags()
	assertEqual(t, 0, int(flags.Ints[0]))
	assertEqualF(t, 0, g.Edge(0, 2).GetDistance(), 1e-6)
}

// --- Wave 3: Advanced tests ---

func TestPillarNodes(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	na := g.GetNodeAccess()
	na.SetNode(0, 0.01, 0.01, 0)
	na.SetNode(4, 0.4, 0.4, 0)
	na.SetNode(14, 0.14, 0.14, 0)
	na.SetNode(10, 0.99, 0.99, 0)

	pointList := util.CreatePointList(1, 1, 1, 2, 1, 3)
	edge := g.Edge(0, 4).SetDistance(100).SetWayGeometry(pointList)
	edge.SetBoolBothDir(tg.carAccessEnc, true, false)

	pointList = util.CreatePointList(1, 5, 1, 6, 1, 7, 1, 8, 1, 9)
	edge = g.Edge(4, 10).SetDistance(100).SetWayGeometry(pointList)
	edge.SetBoolBothDir(tg.carAccessEnc, true, false)

	pointList = util.CreatePointList(1, 13, 1, 12, 1, 11)
	edge = g.Edge(14, 0).SetDistance(100).SetWayGeometry(pointList)
	edge.SetBoolBothDir(tg.carAccessEnc, true, false)

	iter := tg.carAllExplorer.SetBaseNode(0)
	assertTrue(t, iter.Next())
	assertEqual(t, 14, iter.GetAdjNode())
	assertPList(t, util.CreatePointList(1, 11, 1, 12, 1, 13), iter.FetchWayGeometry(util.FetchModePillarOnly))
	assertPList(t, util.CreatePointList(0.01, 0.01, 1, 11, 1, 12, 1, 13), iter.FetchWayGeometry(util.FetchModeBaseAndPillar))
	assertPList(t, util.CreatePointList(1, 11, 1, 12, 1, 13, 0.14, 0.14), iter.FetchWayGeometry(util.FetchModePillarAndAdj))
	assertPList(t, util.CreatePointList(0.01, 0.01, 1, 11, 1, 12, 1, 13, 0.14, 0.14), iter.FetchWayGeometry(util.FetchModeAll))

	assertTrue(t, iter.Next())
	assertEqual(t, 4, iter.GetAdjNode())
	assertPList(t, util.CreatePointList(1, 1, 1, 2, 1, 3), iter.FetchWayGeometry(util.FetchModePillarOnly))
	assertPList(t, util.CreatePointList(0.01, 0.01, 1, 1, 1, 2, 1, 3), iter.FetchWayGeometry(util.FetchModeBaseAndPillar))
	assertPList(t, util.CreatePointList(1, 1, 1, 2, 1, 3, 0.4, 0.4), iter.FetchWayGeometry(util.FetchModePillarAndAdj))
	assertPList(t, util.CreatePointList(0.01, 0.01, 1, 1, 1, 2, 1, 3, 0.4, 0.4), iter.FetchWayGeometry(util.FetchModeAll))

	assertFalse(t, iter.Next())

	iter = tg.carOutExplorer.SetBaseNode(0)
	assertTrue(t, iter.Next())
	assertEqual(t, 4, iter.GetAdjNode())
	assertPList(t, util.CreatePointList(1, 1, 1, 2, 1, 3), iter.FetchWayGeometry(util.FetchModePillarOnly))
	assertFalse(t, iter.Next())

	iter = tg.carInExplorer.SetBaseNode(10)
	assertTrue(t, iter.Next())
	assertEqual(t, 4, iter.GetAdjNode())
	assertPList(t, util.CreatePointList(1, 9, 1, 8, 1, 7, 1, 6, 1, 5), iter.FetchWayGeometry(util.FetchModePillarOnly))
	assertPList(t, util.CreatePointList(0.99, 0.99, 1, 9, 1, 8, 1, 7, 1, 6, 1, 5), iter.FetchWayGeometry(util.FetchModeBaseAndPillar))
	assertPList(t, util.CreatePointList(1, 9, 1, 8, 1, 7, 1, 6, 1, 5, 0.4, 0.4), iter.FetchWayGeometry(util.FetchModePillarAndAdj))
	assertPList(t, util.CreatePointList(0.99, 0.99, 1, 9, 1, 8, 1, 7, 1, 6, 1, 5, 0.4, 0.4), iter.FetchWayGeometry(util.FetchModeAll))
	assertFalse(t, iter.Next())
}

func TestEnabledElevation(t *testing.T) {
	tg := newTestGraph3D(t, true)
	g := tg.graph
	na := g.GetNodeAccess()
	assertTrue(t, na.Is3D())
	na.SetNode(0, 10, 20, -10)
	na.SetNode(1, 11, 2, 100)
	assertEqualF(t, -10, na.GetEle(0), 0.1)
	assertEqualF(t, 100, na.GetEle(1), 0.1)

	g.Edge(0, 1).SetWayGeometry(util.CreatePointList3D(10, 27, 72, 11, 20, 1))
	assertPList3D(t, util.CreatePointList3D(10, 27, 72, 11, 20, 1),
		getEdge(g, 0, 1).FetchWayGeometry(util.FetchModePillarOnly))
	assertPList3D(t, util.CreatePointList3D(10, 20, -10, 10, 27, 72, 11, 20, 1, 11, 2, 100),
		getEdge(g, 0, 1).FetchWayGeometry(util.FetchModeAll))
	assertPList3D(t, util.CreatePointList3D(11, 2, 100, 11, 20, 1, 10, 27, 72, 10, 20, -10),
		getEdge(g, 1, 0).FetchWayGeometry(util.FetchModeAll))
}

func TestDontGrowOnUpdate(t *testing.T) {
	tg := newTestGraph3D(t, true)
	g := tg.graph
	na := g.GetNodeAccess()
	assertTrue(t, na.Is3D())
	na.SetNode(0, 10, 10, 0)
	na.SetNode(1, 11, 20, 1)
	na.SetNode(2, 12, 12, 0.4)

	iter2 := g.Edge(0, 1).SetDistance(100).SetBoolBothDir(tg.carAccessEnc, true, true)
	assertEqual(t, 1, int(g.GetMaxGeoRef()))

	iter2.SetWayGeometry(util.CreatePointList3D(1, 2, 3, 3, 4, 5, 5, 6, 7, 7, 8, 9))
	assertEqual(t, 1+3+4*11, int(g.GetMaxGeoRef()))
	iter2.SetWayGeometry(util.CreatePointList3D(1, 2, 3, 3, 4, 5, 5, 6, 7))
	assertEqual(t, 1+3+4*11, int(g.GetMaxGeoRef()))
	iter2.SetWayGeometry(util.CreatePointList3D(1, 2, 3, 3, 4, 5))
	assertEqual(t, 1+3+4*11, int(g.GetMaxGeoRef()))
	iter2.SetWayGeometry(util.CreatePointList3D(1, 2, 3))
	assertEqual(t, 1+3+4*11, int(g.GetMaxGeoRef()))

	assertPanics(t, func() { iter2.SetWayGeometry(util.CreatePointList3D(1.5, 1, 0, 2, 3, 0)) })
	assertEqual(t, 1+3+4*11, int(g.GetMaxGeoRef()))

	iter1 := g.Edge(0, 2).SetDistance(200).SetBoolBothDir(tg.carAccessEnc, true, true)
	_ = iter1
	iter1.SetWayGeometry(util.CreatePointList3D(3.5, 4.5, 0, 5, 6, 0))
	assertEqual(t, 1+3+4*11+(3+2*11), int(g.GetMaxGeoRef()))
}

func TestDetachEdge(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	g.Edge(0, 1).SetDistance(2).SetBoolBothDir(tg.carAccessEnc, true, true)
	g.Edge(0, 2).SetDistance(2).SetBoolBothDir(tg.carAccessEnc, true, true).
		SetWayGeometry(util.CreatePointList(1, 2, 3, 4)).
		SetBoolBothDir(tg.carAccessEnc, true, false)
	g.Edge(1, 2).SetDistance(2).SetBoolBothDir(tg.carAccessEnc, true, true)

	iter := g.CreateEdgeExplorer(routingutil.AllEdges).SetBaseNode(0)

	// detach before next should panic
	assertPanics(t, func() { iter.Detach(false) })

	iter.Next()
	edgeState02 := iter.Detach(false)
	assertEqual(t, 2, iter.GetAdjNode())
	assertEqualF(t, 1, edgeState02.FetchWayGeometry(util.FetchModePillarOnly).GetLat(0), 0.1)
	assertEqual(t, 2, edgeState02.GetAdjNode())
	assertTrue(t, edgeState02.GetBool(tg.carAccessEnc))

	edgeState20 := iter.Detach(true)
	assertEqual(t, 0, edgeState20.GetAdjNode())
	assertEqual(t, 2, edgeState20.GetBaseNode())
	assertEqualF(t, 3, edgeState20.FetchWayGeometry(util.FetchModePillarOnly).GetLat(0), 0.1)
	assertFalse(t, edgeState20.GetBool(tg.carAccessEnc))

	iter.Next()
	assertEqual(t, 1, iter.GetAdjNode())
	assertEqual(t, 2, edgeState02.GetAdjNode())
	assertEqual(t, 2, edgeState20.GetBaseNode())

	assertEqual(t, 0, iter.FetchWayGeometry(util.FetchModePillarOnly).Size())
	assertEqualF(t, 1, edgeState02.FetchWayGeometry(util.FetchModePillarOnly).GetLat(0), 0.1)
	assertEqualF(t, 3, edgeState20.FetchWayGeometry(util.FetchModePillarOnly).GetLat(0), 0.1)
}

// --- BaseGraphTest-specific tests ---

func TestIdentical(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	assertEqual(t, g.GetNodes(), g.GetBaseGraph().GetNodes())
	assertEqual(t, g.GetEdges(), g.GetBaseGraph().GetEdges())
}

func TestEdgeKey(t *testing.T) {
	tg := newTestGraph(t)
	g := tg.graph
	g.Edge(0, 1).SetDistance(10).SetBoolBothDir(tg.carAccessEnc, true, true)

	// storage direction: 0->1
	edge := g.GetEdgeIteratorState(0, 1)
	assertEqual(t, 0, edge.GetBaseNode())
	assertEqual(t, 1, edge.GetAdjNode())
	edgeKey := edge.GetEdgeKey()
	assertEqual(t, 0, edgeKey) // even = forward

	fromKey := g.GetEdgeIteratorStateForKey(edgeKey)
	assertEqual(t, 0, fromKey.GetBaseNode())
	assertEqual(t, 1, fromKey.GetAdjNode())

	// reverse direction: 1->0
	edge = g.GetEdgeIteratorState(0, 0)
	assertEqual(t, 1, edge.GetBaseNode())
	assertEqual(t, 0, edge.GetAdjNode())
	edgeKey = edge.GetEdgeKey()
	assertEqual(t, 1, edgeKey) // odd = reverse

	fromKey = g.GetEdgeIteratorStateForKey(edgeKey)
	assertEqual(t, 1, fromKey.GetBaseNode())
	assertEqual(t, 0, fromKey.GetAdjNode())
}

func TestGeoRef(t *testing.T) {
	g := NewBaseGraphBuilder(4).CreateGraph()
	defer g.Close()
	g.Edge(0, 1)
	g.Edge(1, 2)

	store := g.Store
	ptr0 := store.ToEdgePointer(0)
	ptr1 := store.ToEdgePointer(1)

	store.SetGeoRef(ptr0, 123)
	assertEqual(t, 123, int(store.GetGeoRef(ptr0)))

	store.SetGeoRef(ptr0, -123)
	assertEqual(t, -123, int(store.GetGeoRef(ptr0)))

	store.SetGeoRef(ptr1, int64(1)<<38)
	assertEqual(t, int(int64(1)<<38), int(store.GetGeoRef(ptr1)))
}

func Test8AndMoreBytesForEdgeFlags(t *testing.T) {
	access0Enc := ev.NewSimpleBooleanEncodedValueDir("car0_access", true)
	speed0Enc := ev.NewDecimalEncodedValueImplFull("car0_speed", 29, 0, 0.001, false, false, false)
	access1Enc := ev.NewSimpleBooleanEncodedValueDir("car1_access", true)
	speed1Enc := ev.NewDecimalEncodedValueImplFull("car1_speed", 29, 0, 0.001, false, false, false)

	cfg := ev.NewInitializerConfig()
	access0Enc.Init(cfg)
	speed0Enc.Init(cfg)
	access1Enc.Init(cfg)
	speed1Enc.Init(cfg)

	bytesForFlags := cfg.GetRequiredBytes()
	g := NewBaseGraphBuilder(bytesForFlags).CreateGraph()

	edge := g.Edge(0, 1)
	impl := edge.(*EdgeIteratorStateImpl)
	intsRef := g.Store.CreateEdgeFlags()
	intsRef.Ints[0] = math.MaxInt32 / 3
	impl.SetFlags(intsRef)
	readBack := impl.GetFlags()
	assertEqual(t, int(math.MaxInt32/3), int(readBack.Ints[0]))
	g.Close()

	g = NewBaseGraphBuilder(bytesForFlags).CreateGraph()
	defer g.Close()

	edge = g.Edge(0, 1)
	util.SetSpeed(99.123, true, true, access0Enc, speed0Enc, edge)
	assertEqualF(t, 99.123, edge.GetDecimal(speed0Enc), 1e-3)

	edgeIter := getEdge(g, 1, 0)
	assertEqualF(t, 99.123, edgeIter.GetDecimal(speed0Enc), 1e-3)
	assertTrue(t, edgeIter.GetBool(access0Enc))
	assertTrue(t, edgeIter.GetReverseBool(access0Enc))

	edge = g.Edge(2, 3)
	util.SetSpeed(44.123, true, false, access1Enc, speed1Enc, edge)
	assertEqualF(t, 44.123, edge.GetDecimal(speed1Enc), 1e-3)

	edgeIter = getEdge(g, 3, 2)
	assertEqualF(t, 44.123, edgeIter.GetDecimal(speed1Enc), 1e-3)
	assertEqualF(t, 44.123, edgeIter.GetReverseDecimal(speed1Enc), 1e-3)
	assertFalse(t, edgeIter.GetBool(access1Enc))
	assertTrue(t, edgeIter.GetReverseBool(access1Enc))
}
