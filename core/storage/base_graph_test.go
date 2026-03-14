package storage

import (
	"math"
	"testing"

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
