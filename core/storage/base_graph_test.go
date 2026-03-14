package storage

import (
	"math"
	"testing"

	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/util"
)

func TestBaseGraph_SaveAndLoadFileFormat(t *testing.T) {
	dir := testDir(t)

	// Create graph with 3 nodes and 2 edges
	d := NewRAMDirectory(dir, true).Init().(*GHDirectory)
	g := NewBaseGraph(d, false, false, -1, 4)
	g.Create(100)

	na := g.GetNodeAccess()
	na.SetNode(0, 52.53, 13.35, 0)
	na.SetNode(1, 52.50, 13.40, 0)
	na.SetNode(2, 52.51, 13.38, 0)

	g.Edge(0, 1).SetDistance(1234.5)
	g.Edge(1, 2).SetDistance(567.8)

	g.Flush()
	g.Close()
	d.Close()

	// Reload and verify
	d2 := NewRAMDirectory(dir, true).Init().(*GHDirectory)
	g2 := NewBaseGraph(d2, false, false, -1, 4)
	if !g2.LoadExisting() {
		t.Fatal("expected LoadExisting to return true")
	}
	if g2.GetNodes() != 3 {
		t.Fatalf("expected 3 nodes, got %d", g2.GetNodes())
	}
	if g2.GetEdges() != 2 {
		t.Fatalf("expected 2 edges, got %d", g2.GetEdges())
	}

	na2 := g2.GetNodeAccess()
	if math.Abs(na2.GetLat(0)-52.53) > 1e-4 {
		t.Fatalf("node 0 lat: expected ~52.53, got %f", na2.GetLat(0))
	}
	if math.Abs(na2.GetLon(0)-13.35) > 1e-4 {
		t.Fatalf("node 0 lon: expected ~13.35, got %f", na2.GetLon(0))
	}
	if math.Abs(na2.GetLat(1)-52.50) > 1e-4 {
		t.Fatalf("node 1 lat: expected ~52.50, got %f", na2.GetLat(1))
	}

	if math.Abs(g2.GetDist(0)-1234.5) > 0.1 {
		t.Fatalf("edge 0 dist: expected ~1234.5, got %f", g2.GetDist(0))
	}
	if math.Abs(g2.GetDist(1)-567.8) > 0.1 {
		t.Fatalf("edge 1 dist: expected ~567.8, got %f", g2.GetDist(1))
	}

	bounds := g2.GetBounds()
	if !bounds.IsValid() {
		t.Fatal("expected valid bounds after reload")
	}
	g2.Close()
	d2.Close()
}

func TestBaseGraph_Bounds(t *testing.T) {
	g := NewBaseGraphBuilder(4).CreateGraph()
	na := g.GetNodeAccess()
	na.SetNode(0, 52.53, 13.35, 0)
	na.SetNode(1, 48.15, 11.58, 0)

	bounds := g.GetBounds()
	if math.Abs(bounds.MinLat-48.15) > 1e-4 {
		t.Fatalf("expected minLat ~48.15, got %f", bounds.MinLat)
	}
	if math.Abs(bounds.MaxLat-52.53) > 1e-4 {
		t.Fatalf("expected maxLat ~52.53, got %f", bounds.MaxLat)
	}
	g.Close()
}

func TestBaseGraph_NodeAccess(t *testing.T) {
	g := NewBaseGraphBuilder(4).SetWithElevation(true).CreateGraph()
	na := g.GetNodeAccess()
	na.SetNode(0, 52.53, 13.35, 100.5)
	na.SetNode(1, 48.15, 11.58, 200.7)

	if math.Abs(na.GetLat(0)-52.53) > 1e-4 {
		t.Fatalf("lat: expected ~52.53, got %f", na.GetLat(0))
	}
	if math.Abs(na.GetLon(0)-13.35) > 1e-4 {
		t.Fatalf("lon: expected ~13.35, got %f", na.GetLon(0))
	}
	if math.Abs(na.GetEle(0)-100.5) > 0.1 {
		t.Fatalf("ele: expected ~100.5, got %f", na.GetEle(0))
	}
	if !na.Is3D() {
		t.Fatal("expected 3D to be true")
	}
	g.Close()
}

func TestBaseGraph_SaveAndFreeze(t *testing.T) {
	dir := testDir(t)
	d := NewRAMDirectory(dir, true).Init().(*GHDirectory)
	g := NewBaseGraph(d, true, false, -1, 4)
	g.Create(100)
	g.Edge(1, 0)
	g.Freeze()

	g.Flush()
	g.Close()
	d.Close()

	d2 := NewRAMDirectory(dir, true).Init().(*GHDirectory)
	g2 := NewBaseGraph(d2, true, false, -1, 4)
	if !g2.LoadExisting() {
		t.Fatal("expected LoadExisting true")
	}
	if g2.GetNodes() != 2 {
		t.Fatalf("expected 2 nodes, got %d", g2.GetNodes())
	}
	if !g2.IsFrozen() {
		t.Fatal("expected graph to be frozen after reload")
	}
	g2.Close()
	d2.Close()
}

func TestBaseGraph_DimMismatch(t *testing.T) {
	dir := testDir(t)

	// Create with 2D
	d := NewRAMDirectory(dir, true).Init().(*GHDirectory)
	g := NewBaseGraph(d, false, false, -1, 4)
	g.Create(100)
	g.Flush()
	g.Close()
	d.Close()

	// Try to load as 3D — should panic
	d2 := NewRAMDirectory(dir, true).Init().(*GHDirectory)
	g2 := NewBaseGraph(d2, true, false, -1, 4)
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on dimension mismatch")
		}
		g2.Close()
		d2.Close()
	}()
	g2.LoadExisting()
}

func TestBaseGraph_OutOfBounds(t *testing.T) {
	g := NewBaseGraphBuilder(4).CreateGraph()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for out-of-bounds edge access")
		}
		g.Close()
	}()
	g.GetEdgeIteratorState(0, math.MinInt32)
}

func TestBaseGraph_SetGetFlagsRaw(t *testing.T) {
	g := NewBaseGraphBuilder(1).CreateGraph()
	edge := g.Edge(0, 1).(*EdgeIteratorStateImpl)
	flags := NewIntsRef(1)
	flags.Ints[0] = 10
	edge.SetFlags(flags)
	if edge.GetFlags().Ints[0] != 10 {
		t.Fatalf("expected flags 10, got %d", edge.GetFlags().Ints[0])
	}
	flags.Ints[0] = 9
	edge.SetFlags(flags)
	if edge.GetFlags().Ints[0] != 9 {
		t.Fatalf("expected flags 9, got %d", edge.GetFlags().Ints[0])
	}
	g.Close()
}

func TestBaseGraph_EdgeKey(t *testing.T) {
	g := NewBaseGraphBuilder(4).CreateGraph()
	g.Edge(0, 1).SetDistance(10)

	// storage direction
	e0 := g.GetEdgeIteratorState(0, math.MinInt32)
	if e0.GetBaseNode() != 0 || e0.GetAdjNode() != 1 {
		t.Fatalf("storage direction: expected 0→1, got %d→%d", e0.GetBaseNode(), e0.GetAdjNode())
	}
	if e0.GetBool(util.ReverseState) {
		t.Fatal("storage direction should not be reversed")
	}
	if e0.GetEdge() != 0 || e0.GetEdgeKey() != 0 {
		t.Fatalf("expected edge=0, key=0, got edge=%d, key=%d", e0.GetEdge(), e0.GetEdgeKey())
	}

	// reverse direction
	e0r := g.GetEdgeIteratorState(0, 0)
	if e0r.GetBaseNode() != 1 || e0r.GetAdjNode() != 0 {
		t.Fatalf("reverse direction: expected 1→0, got %d→%d", e0r.GetBaseNode(), e0r.GetAdjNode())
	}
	if !e0r.GetBool(util.ReverseState) {
		t.Fatal("reverse direction should be reversed")
	}
	if e0r.GetEdge() != 0 || e0r.GetEdgeKey() != 1 {
		t.Fatalf("expected edge=0, key=1, got edge=%d, key=%d", e0r.GetEdge(), e0r.GetEdgeKey())
	}

	// from edge key
	ek0 := g.GetEdgeIteratorStateForKey(0)
	if ek0.GetBaseNode() != 0 || ek0.GetAdjNode() != 1 || ek0.GetEdgeKey() != 0 {
		t.Fatal("edge key 0 mismatch")
	}
	ek1 := g.GetEdgeIteratorStateForKey(1)
	if ek1.GetBaseNode() != 1 || ek1.GetAdjNode() != 0 || ek1.GetEdgeKey() != 1 {
		t.Fatal("edge key 1 mismatch")
	}
	g.Close()
}

func TestBaseGraph_GeoRef(t *testing.T) {
	g := NewBaseGraphBuilder(4).CreateGraph()
	g.Edge(0, 1) // need at least one edge to have edge pointers
	ne := g.Store
	ep := ne.ToEdgePointer(0)

	ne.SetGeoRef(ep, 123)
	if ne.GetGeoRef(ep) != 123 {
		t.Fatalf("expected 123, got %d", ne.GetGeoRef(ep))
	}
	ne.SetGeoRef(ep, -123)
	if ne.GetGeoRef(ep) != -123 {
		t.Fatalf("expected -123, got %d", ne.GetGeoRef(ep))
	}
	ne.SetGeoRef(ep, 1<<38)
	if ne.GetGeoRef(ep) != 1<<38 {
		t.Fatalf("expected 1<<38, got %d", ne.GetGeoRef(ep))
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for geoRef 1<<39")
		}
		g.Close()
	}()
	ne.SetGeoRef(ep, 1<<39)
}

func TestBaseGraph_WithTurnCosts(t *testing.T) {
	dir := testDir(t)
	d := NewRAMDirectory(dir, true).Init().(*GHDirectory)
	g := NewBaseGraph(d, false, true, -1, 4)
	g.Create(100)

	na := g.GetNodeAccess()
	na.SetNode(0, 52.53, 13.35, 0)
	na.SetNode(1, 52.50, 13.40, 0)
	na.SetNode(2, 52.51, 13.38, 0)

	g.Edge(0, 1)
	g.Edge(1, 2)

	// Set a turn cost entry
	idx := g.TurnCostStorage.FindOrCreateEntry(na, 0, 1, 1)
	g.TurnCostStorage.SetFlags(idx, 42)

	g.Flush()
	g.Close()
	d.Close()

	// Reload
	d2 := NewRAMDirectory(dir, true).Init().(*GHDirectory)
	g2 := NewBaseGraph(d2, false, true, -1, 4)
	if !g2.LoadExisting() {
		t.Fatal("expected LoadExisting true")
	}
	if g2.TurnCostStorage.Count() != 1 {
		t.Fatalf("expected 1 turn cost entry, got %d", g2.TurnCostStorage.Count())
	}
	g2.Close()
	d2.Close()
}

func newTCEncodingManager(t *testing.T) (ev.BooleanEncodedValue, ev.DecimalEncodedValue, ev.DecimalEncodedValue, int) {
	t.Helper()
	carAccessEnc := ev.NewSimpleBooleanEncodedValueDir("car_access", true)
	carSpeedEnc := ev.NewDecimalEncodedValueImpl("car_speed", 5, 5, false)
	turnCostEnc := ev.TurnCostCreate("car", 1400)
	em := routingutil.Start().
		Add(carAccessEnc).Add(carSpeedEnc).
		AddTurnCostEncodedValue(turnCostEnc).Build()
	return carAccessEnc, carSpeedEnc, turnCostEnc, em.BytesForFlags
}

func TestBaseGraph_WithTurnCosts_SaveAndFileFormat(t *testing.T) {
	carAccessEnc, _, turnCostEnc, bytesForFlags := newTCEncodingManager(t)

	dir := testDir(t)
	d := NewRAMDirectory(dir, true).Init().(*GHDirectory)
	g := NewBaseGraph(d, true, true, -1, bytesForFlags)
	g.Create(100)

	na := g.GetNodeAccess()
	if !na.Is3D() {
		t.Fatal("expected 3D")
	}
	na.SetNode(0, 10, 10, 0)
	na.SetNode(1, 11, 20, 1)
	na.SetNode(2, 12, 12, 0.4)

	iter2 := g.Edge(0, 1).SetDistance(100).SetBoolBothDir(carAccessEnc, true, true)
	iter2.SetWayGeometry(util.CreatePointList3D(1.5, 1, 0, 2, 3, 0))
	iter1 := g.Edge(0, 2).SetDistance(200).SetBoolBothDir(carAccessEnc, true, true)
	iter1.SetWayGeometry(util.CreatePointList3D(3.5, 4.5, 0, 5, 6, 0))
	g.Edge(9, 10).SetDistance(200).SetBoolBothDir(carAccessEnc, true, true)
	g.Edge(9, 11).SetDistance(200).SetBoolBothDir(carAccessEnc, true, true)
	g.Edge(1, 2).SetDistance(120).SetBoolBothDir(carAccessEnc, true, false)

	tc := g.GetTurnCostStorage()
	tc.SetDecimal(na, turnCostEnc, iter1.GetEdge(), 0, iter2.GetEdge(), 1337)
	tc.SetDecimal(na, turnCostEnc, iter2.GetEdge(), 0, iter1.GetEdge(), 666)
	tc.SetDecimal(na, turnCostEnc, iter1.GetEdge(), 1, iter2.GetEdge(), 815)

	iter1.SetKeyValues(map[string]any{"name": "named street1"})
	iter2.SetKeyValues(map[string]any{"name": "named street2"})

	g.Flush()
	g.Close()
	d.Close()

	// Reload
	d2 := NewRAMDirectory(dir, true).Init().(*GHDirectory)
	g2 := NewBaseGraph(d2, true, true, -1, bytesForFlags)
	if !g2.LoadExisting() {
		t.Fatal("expected LoadExisting true")
	}
	if g2.GetNodes() != 12 {
		t.Fatalf("expected 12 nodes, got %d", g2.GetNodes())
	}

	// street names persist
	e1 := g2.GetEdgeIteratorState(iter1.GetEdge(), iter1.GetAdjNode())
	e2 := g2.GetEdgeIteratorState(iter2.GetEdge(), iter2.GetAdjNode())
	if e1.GetName() != "named street1" {
		t.Fatalf("expected 'named street1', got %q", e1.GetName())
	}
	if e2.GetName() != "named street2" {
		t.Fatalf("expected 'named street2', got %q", e2.GetName())
	}

	// turn costs persist
	na2 := g2.GetNodeAccess()
	tc2 := g2.GetTurnCostStorage()
	if v := tc2.GetDecimal(na2, turnCostEnc, iter1.GetEdge(), 0, iter2.GetEdge()); math.Abs(v-1337) > 0.1 {
		t.Fatalf("expected turn cost 1337, got %f", v)
	}
	if v := tc2.GetDecimal(na2, turnCostEnc, iter2.GetEdge(), 0, iter1.GetEdge()); math.Abs(v-666) > 0.1 {
		t.Fatalf("expected turn cost 666, got %f", v)
	}
	if v := tc2.GetDecimal(na2, turnCostEnc, iter1.GetEdge(), 1, iter2.GetEdge()); math.Abs(v-815) > 0.1 {
		t.Fatalf("expected turn cost 815, got %f", v)
	}
	// non-existent turn cost returns 0
	if v := tc2.GetDecimal(na2, turnCostEnc, iter1.GetEdge(), 3, iter2.GetEdge()); v != 0 {
		t.Fatalf("expected 0 for non-existent turn cost, got %f", v)
	}

	g2.Edge(3, 4).SetDistance(123).SetBoolBothDir(carAccessEnc, true, true).
		SetWayGeometry(util.CreatePointList3D(4.4, 5.5, 0, 6.6, 7.7, 0))

	g2.Close()
	d2.Close()
}

func TestBaseGraph_WithTurnCosts_EnsureCapacity(t *testing.T) {
	carAccessEnc, _, turnCostEnc, bytesForFlags := newTCEncodingManager(t)

	dir := testDir(t)
	d := NewRAMDirectory(dir, true).Init().(*GHDirectory)
	g := NewBaseGraph(d, false, true, 128, bytesForFlags)
	g.Create(100)

	tc := g.GetTurnCostStorage()
	if tc.GetCapacity() != 128 {
		t.Fatalf("expected initial capacity 128, got %d", tc.GetCapacity())
	}

	na := g.GetNodeAccess()
	for i := 0; i < 100; i++ {
		na.SetNode(i, float64(i)*0.9, float64(i)*1.8, 0)
	}

	// Make node 50 the 'center' node
	for nodeID := 51; nodeID < 100; nodeID++ {
		g.Edge(50, nodeID).SetDistance(float64(nodeID)).SetBoolBothDir(carAccessEnc, true, true)
	}
	for nodeID := 0; nodeID < 50; nodeID++ {
		g.Edge(nodeID, 50).SetDistance(float64(nodeID)).SetBoolBothDir(carAccessEnc, true, true)
	}

	// add turn cost entries around node 50
	for edgeID := 0; edgeID < 52; edgeID++ {
		tc.SetDecimal(na, turnCostEnc, edgeID, 50, edgeID+50, 1337)
		tc.SetDecimal(na, turnCostEnc, edgeID+50, 50, edgeID, 1337)
	}

	if tc.GetCapacity()/16 != 104 {
		t.Fatalf("expected 104 entries capacity, got %d", tc.GetCapacity()/16)
	}

	tc.SetDecimal(na, turnCostEnc, 0, 50, 2, 1337)
	// A new segment should be added: 128/16 = 8 more entries
	if tc.GetCapacity()/16 != 112 {
		t.Fatalf("expected 112 entries capacity after growth, got %d", tc.GetCapacity()/16)
	}

	g.Close()
	d.Close()
}

func TestBaseGraph_WithTurnCosts_InitializeTurnCost(t *testing.T) {
	_, _, _, bytesForFlags := newTCEncodingManager(t)

	g := NewBaseGraphBuilder(bytesForFlags).SetWithElevation(true).SetWithTurnCosts(true).CreateGraph()
	na := g.GetNodeAccess()

	// turn cost index is initialized to NoTurnEntry
	na.SetNode(4001, 10, 11, 10)
	if na.GetTurnCostIndex(4001) != NoTurnEntry {
		t.Fatalf("expected NoTurnEntry for new node, got %d", na.GetTurnCostIndex(4001))
	}

	na.SetNode(4000, 10, 11, 10)
	na.SetTurnCostIndex(4000, 12)
	// updating elevation should not alter turn cost index
	na.SetNode(4000, 10, 11, 11)
	if na.GetTurnCostIndex(4000) != 12 {
		t.Fatalf("expected turn cost index 12 after elevation update, got %d", na.GetTurnCostIndex(4000))
	}

	g.Close()
}
