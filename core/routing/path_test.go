package routing

import (
	"math"
	"testing"

	"gohopper/core/routing/ev"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// --- test helpers ---

// testSetup creates a graph with a speed encoded value initialized and ready to use.
func testSetup(t *testing.T) (*storage.BaseGraph, ev.DecimalEncodedValue) {
	t.Helper()
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	cfg := ev.NewInitializerConfig()
	speedEnc.Init(cfg)
	bytesForFlags := cfg.GetRequiredBytes()
	g := storage.NewBaseGraphBuilder(bytesForFlags).CreateGraph()
	t.Cleanup(func() { g.Close() })
	return g, speedEnc
}

// extractPath is a helper matching the Java PathTest.extractPath convenience method.
func extractPath(graph storage.Graph, w weighting.Weighting, sptEntry *SPTEntry) *Path {
	return ExtractPath(graph, w, sptEntry)
}

// --- tests ---

// TestPath_Defaults mirrors Java PathTest.testFound.
func TestPath_Defaults(t *testing.T) {
	g, _ := testSetup(t)
	p := NewPath(g)

	if p.Found {
		t.Fatal("expected Found to be false for a new path")
	}
	if math.Abs(p.Distance) > 1e-7 {
		t.Fatalf("expected Distance ~0, got %v", p.Distance)
	}
	if p.Weight != math.MaxFloat64 {
		t.Fatalf("expected Weight to be MaxFloat64, got %v", p.Weight)
	}
	nodes := p.CalcNodes()
	if len(nodes) != 0 {
		t.Fatalf("expected empty CalcNodes, got %v", nodes)
	}
}

// TestPath_AddEdge verifies adding edges and GetEdgeCount.
func TestPath_AddEdge(t *testing.T) {
	g, _ := testSetup(t)
	p := NewPath(g)

	p.AddEdge(10)
	p.AddEdge(20)
	p.AddEdge(30)

	if p.GetEdgeCount() != 3 {
		t.Fatalf("expected 3 edges, got %d", p.GetEdgeCount())
	}
	if p.EdgeIDs[0] != 10 || p.EdgeIDs[1] != 20 || p.EdgeIDs[2] != 30 {
		t.Fatalf("unexpected EdgeIDs: %v", p.EdgeIDs)
	}
}

// TestPath_SettersChaining verifies chainable setters.
func TestPath_SettersChaining(t *testing.T) {
	g, _ := testSetup(t)
	p := NewPath(g)

	result := p.SetWeight(42.0).SetDistance(100.0).SetTime(5000).SetFromNode(1).SetEndNode(2).SetFound(true)

	if result != p {
		t.Fatal("chaining should return the same Path pointer")
	}
	if p.Weight != 42.0 {
		t.Fatalf("expected Weight=42.0, got %v", p.Weight)
	}
	if p.Distance != 100.0 {
		t.Fatalf("expected Distance=100.0, got %v", p.Distance)
	}
	if p.Time != 5000 {
		t.Fatalf("expected Time=5000, got %d", p.Time)
	}
	if p.FromNode != 1 {
		t.Fatalf("expected FromNode=1, got %d", p.FromNode)
	}
	if p.EndNode != 2 {
		t.Fatalf("expected EndNode=2, got %d", p.EndNode)
	}
	if !p.Found {
		t.Fatal("expected Found=true")
	}
}

// TestPath_CalcNodes builds a path with edges and verifies the node list.
func TestPath_CalcNodes(t *testing.T) {
	g, speedEnc := testSetup(t)
	na := g.GetNodeAccess()

	// Create a simple 3-node graph: 0 -- 1 -- 2
	na.SetNode(0, 0.0, 0.0, 0)
	na.SetNode(1, 1.0, 0.0, 0)
	na.SetNode(2, 2.0, 0.0, 0)

	edge01 := g.Edge(0, 1).SetDistance(1000)
	edge01.SetDecimalBothDir(speedEnc, 50.0, 50.0)
	edge12 := g.Edge(1, 2).SetDistance(2000)
	edge12.SetDecimalBothDir(speedEnc, 50.0, 50.0)

	p := NewPath(g)
	p.SetFromNode(0)
	p.SetEndNode(2)
	p.SetFound(true)
	p.AddEdge(edge01.GetEdge())
	p.AddEdge(edge12.GetEdge())

	nodes := p.CalcNodes()
	expected := []int{0, 1, 2}
	if len(nodes) != len(expected) {
		t.Fatalf("expected %d nodes, got %d", len(expected), len(nodes))
	}
	for i, n := range expected {
		if nodes[i] != n {
			t.Fatalf("at index %d: expected node %d, got %d", i, n, nodes[i])
		}
	}
}

// TestPath_CalcNodes_EmptyFoundPath tests that an empty found path returns just the end node.
func TestPath_CalcNodes_EmptyFoundPath(t *testing.T) {
	g, _ := testSetup(t)
	na := g.GetNodeAccess()
	na.SetNode(0, 0.0, 0.0, 0)

	p := NewPath(g)
	p.SetEndNode(0)
	p.SetFound(true)

	nodes := p.CalcNodes()
	if len(nodes) != 1 || nodes[0] != 0 {
		t.Fatalf("expected [0], got %v", nodes)
	}
}

// TestExtractPath builds an SPTEntry chain, extracts a path, and verifies distance/time/nodes/found.
// This mirrors the core of Java PathTest.testWayList.
func TestExtractPath(t *testing.T) {
	g, speedEnc := testSetup(t)
	na := g.GetNodeAccess()

	na.SetNode(0, 0.0, 0.1, 0)
	na.SetNode(1, 1.0, 0.1, 0)
	na.SetNode(2, 2.0, 0.1, 0)

	edge1 := g.Edge(0, 1).SetDistance(1000)
	edge1.SetDecimalBothDir(speedEnc, 10.0, 10.0)

	edge2 := g.Edge(2, 1).SetDistance(2000)
	edge2.SetDecimalBothDir(speedEnc, 50.0, 50.0)

	// SPTEntry chain: root(node=0) -> edge1 to node 1 -> edge2 to node 2
	root := NewSPTEntry(0, 1)
	mid := NewSPTEntryFull(edge1.GetEdge(), 1, 1, root)
	leaf := NewSPTEntryFull(edge2.GetEdge(), 2, 1, mid)

	w := weighting.NewSpeedWeighting(speedEnc)
	path := extractPath(g, w, leaf)

	// Path should be found
	if !path.Found {
		t.Fatal("expected path to be found")
	}

	// Check distance = 1000 + 2000 = 3000
	if math.Abs(path.Distance-3000.0) > 1e-7 {
		t.Fatalf("expected distance 3000.0, got %v", path.Distance)
	}

	// Check nodes: 0 -> 1 -> 2
	nodes := path.CalcNodes()
	expectedNodes := []int{0, 1, 2}
	if len(nodes) != len(expectedNodes) {
		t.Fatalf("expected %d nodes, got %d: %v", len(expectedNodes), len(nodes), nodes)
	}
	for i, n := range expectedNodes {
		if nodes[i] != n {
			t.Fatalf("at index %d: expected node %d, got %d", i, n, nodes[i])
		}
	}

	// Check time:
	// edge1: dist=1000, speed=10 -> weight=100 -> millis=100000
	// edge2: dist=2000, speed=50 -> weight=40  -> millis=40000
	// total: 140000 ms
	if path.Time != 140000 {
		t.Fatalf("expected time 140000, got %d", path.Time)
	}

	// Check edge count
	if path.GetEdgeCount() != 2 {
		t.Fatalf("expected 2 edges, got %d", path.GetEdgeCount())
	}
}

// TestExtractPath_NilSPTEntry verifies that a nil sptEntry produces an empty unfound path.
func TestExtractPath_NilSPTEntry(t *testing.T) {
	g, speedEnc := testSetup(t)
	w := weighting.NewSpeedWeighting(speedEnc)

	path := extractPath(g, w, nil)

	if path.Found {
		t.Fatal("expected path to not be found for nil sptEntry")
	}
	if path.GetEdgeCount() != 0 {
		t.Fatalf("expected 0 edges, got %d", path.GetEdgeCount())
	}
}

// TestExtractPath_SingleNode verifies extraction for a single-node SPTEntry (start == end).
func TestExtractPath_SingleNode(t *testing.T) {
	g, speedEnc := testSetup(t)
	na := g.GetNodeAccess()
	na.SetNode(0, 0.0, 0.0, 0)

	w := weighting.NewSpeedWeighting(speedEnc)
	root := NewSPTEntry(0, 0)

	path := extractPath(g, w, root)

	if !path.Found {
		t.Fatal("expected path to be found")
	}
	if path.GetEdgeCount() != 0 {
		t.Fatalf("expected 0 edges, got %d", path.GetEdgeCount())
	}
	if path.Distance != 0 {
		t.Fatalf("expected distance 0, got %v", path.Distance)
	}
	nodes := path.CalcNodes()
	if len(nodes) != 1 || nodes[0] != 0 {
		t.Fatalf("expected [0], got %v", nodes)
	}
}

// TestExtractPath_CalcPoints verifies that CalcPoints produces the correct geometry.
// This mirrors the geometry assertion from Java PathTest.testWayList.
func TestExtractPath_CalcPoints(t *testing.T) {
	g, speedEnc := testSetup(t)
	na := g.GetNodeAccess()

	na.SetNode(0, 0.0, 0.1, 0)
	na.SetNode(1, 1.0, 0.1, 0)
	na.SetNode(2, 2.0, 0.1, 0)

	edge1 := g.Edge(0, 1).SetDistance(1000)
	edge1.SetDecimalBothDir(speedEnc, 10.0, 10.0)
	edge1.SetWayGeometry(util.CreatePointList(8, 1, 9, 1))

	edge2 := g.Edge(2, 1).SetDistance(2000)
	edge2.SetDecimalBothDir(speedEnc, 50.0, 50.0)
	edge2.SetWayGeometry(util.CreatePointList(11, 1, 10, 1))

	// SPTEntry chain: root(node=0) -> edge1 to node 1 -> edge2 to node 2
	root := NewSPTEntry(0, 1)
	mid := NewSPTEntryFull(edge1.GetEdge(), 1, 1, root)
	leaf := NewSPTEntryFull(edge2.GetEdge(), 2, 1, mid)

	w := weighting.NewSpeedWeighting(speedEnc)
	path := extractPath(g, w, leaf)

	points := path.CalcPoints()

	// Expected: (0, 0.1), (8, 1), (9, 1), (1, 0.1), (10, 1), (11, 1), (2, 0.1)
	expectedPL := util.CreatePointList(0, 0.1, 8, 1, 9, 1, 1, 0.1, 10, 1, 11, 1, 2, 0.1)
	if !points.Equals(expectedPL) {
		t.Fatalf("points mismatch\ngot:      %s\nexpected: %s", points, expectedPL)
	}
}

// TestExtractPath_Reverse verifies extraction in reverse order.
// Mirrors the reverse direction test from Java PathTest.testWayList.
func TestExtractPath_Reverse(t *testing.T) {
	g, speedEnc := testSetup(t)
	na := g.GetNodeAccess()

	na.SetNode(0, 0.0, 0.1, 0)
	na.SetNode(1, 1.0, 0.1, 0)
	na.SetNode(2, 2.0, 0.1, 0)

	edge1 := g.Edge(0, 1).SetDistance(1000)
	edge1.SetDecimalBothDir(speedEnc, 10.0, 10.0)
	edge1.SetWayGeometry(util.CreatePointList(8, 1, 9, 1))

	edge2 := g.Edge(2, 1).SetDistance(2000)
	edge2.SetDecimalBothDir(speedEnc, 50.0, 50.0)
	edge2.SetWayGeometry(util.CreatePointList(11, 1, 10, 1))

	// Reverse order: root(node=2) -> edge2 to node 1 -> edge1 to node 0
	root := NewSPTEntry(2, 1)
	mid := NewSPTEntryFull(edge2.GetEdge(), 1, 1, root)
	leaf := NewSPTEntryFull(edge1.GetEdge(), 0, 1, mid)

	w := weighting.NewSpeedWeighting(speedEnc)
	path := extractPath(g, w, leaf)

	// 2-1-0
	expectedPL := util.CreatePointList(2, 0.1, 11, 1, 10, 1, 1, 0.1, 9, 1, 8, 1, 0, 0.1)
	points := path.CalcPoints()
	if !points.Equals(expectedPL) {
		t.Fatalf("points mismatch\ngot:      %s\nexpected: %s", points, expectedPL)
	}

	nodes := path.CalcNodes()
	expectedNodes := []int{2, 1, 0}
	if len(nodes) != len(expectedNodes) {
		t.Fatalf("expected %d nodes, got %d", len(expectedNodes), len(nodes))
	}
	for i, n := range expectedNodes {
		if nodes[i] != n {
			t.Fatalf("at index %d: expected node %d, got %d", i, n, nodes[i])
		}
	}
}

// TestPath_String verifies the string representation.
func TestPath_String(t *testing.T) {
	g, _ := testSetup(t)
	p := NewPath(g)
	p.SetWeight(42.5).SetDistance(100.0).SetTime(5000).SetFound(true)
	p.AddEdge(1)
	p.AddEdge(2)

	s := p.String()
	expected := "found: true, weight: 42.5, time: 5000, distance: 100, edges: 2"
	if s != expected {
		t.Fatalf("expected %q, got %q", expected, s)
	}
}

// TestPath_AddDistanceAndTime verifies AddDistance and AddTime accumulators.
func TestPath_AddDistanceAndTime(t *testing.T) {
	g, _ := testSetup(t)
	p := NewPath(g)

	p.AddDistance(100.0)
	p.AddDistance(200.0)
	if math.Abs(p.Distance-300.0) > 1e-7 {
		t.Fatalf("expected distance 300.0, got %v", p.Distance)
	}

	p.AddTime(1000)
	p.AddTime(2000)
	if p.Time != 3000 {
		t.Fatalf("expected time 3000, got %d", p.Time)
	}
}

// TestPath_ForEveryEdge verifies the ForEveryEdge callback mechanism.
func TestPath_ForEveryEdge(t *testing.T) {
	g, speedEnc := testSetup(t)
	na := g.GetNodeAccess()

	na.SetNode(0, 0.0, 0.0, 0)
	na.SetNode(1, 1.0, 0.0, 0)
	na.SetNode(2, 2.0, 0.0, 0)

	e01 := g.Edge(0, 1).SetDistance(100)
	e01.SetDecimalBothDir(speedEnc, 10.0, 10.0)
	e12 := g.Edge(1, 2).SetDistance(200)
	e12.SetDecimalBothDir(speedEnc, 10.0, 10.0)

	p := NewPath(g)
	p.SetFromNode(0)
	p.SetEndNode(2)
	p.SetFound(true)
	p.AddEdge(e01.GetEdge())
	p.AddEdge(e12.GetEdge())

	var visitedEdges []int
	var visitedPrevEdges []int
	finishCalled := false
	p.ForEveryEdge(&testEdgeVisitor{
		nextFn: func(edge util.EdgeIteratorState, index int, prevEdgeID int) {
			visitedEdges = append(visitedEdges, edge.GetEdge())
			visitedPrevEdges = append(visitedPrevEdges, prevEdgeID)
		},
		finishFn: func() { finishCalled = true },
	})

	if len(visitedEdges) != 2 {
		t.Fatalf("expected 2 visited edges, got %d", len(visitedEdges))
	}
	if !finishCalled {
		t.Fatal("expected Finish to be called")
	}
	if visitedPrevEdges[0] != util.NoEdge {
		t.Fatalf("first edge should have prevEdge=NoEdge, got %d", visitedPrevEdges[0])
	}
}

type testEdgeVisitor struct {
	nextFn   func(edge util.EdgeIteratorState, index int, prevEdgeID int)
	finishFn func()
}

func (v *testEdgeVisitor) Next(edge util.EdgeIteratorState, index int, prevEdgeID int) {
	v.nextFn(edge, index, prevEdgeID)
}

func (v *testEdgeVisitor) Finish() {
	v.finishFn()
}
