package storage

import (
	"math"
	"testing"
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

	e0 := g.Edge(0, 1)
	g.SetDist(e0, 1234.5)
	e1 := g.Edge(1, 2)
	g.SetDist(e1, 567.8)

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
