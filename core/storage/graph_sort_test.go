package storage

import (
	"math"
	"testing"

	routingutil "gohopper/core/routing/util"
)

func TestSortGraphSimple(t *testing.T) {
	// Create a small graph and verify sort reorders nodes/edges properly.
	g := NewBaseGraphBuilder(4).CreateGraph()
	na := g.GetNodeAccess()

	// Place nodes at different geographic positions so Hilbert sort reorders them.
	// Node 0: far east, Node 1: center, Node 2: far west
	na.SetNode(0, 0, 90, 0)   // far east
	na.SetNode(1, 0, 0, 0)    // center
	na.SetNode(2, 0, -90, 0)  // far west

	g.Edge(0, 1).SetDistance(100)
	g.Edge(1, 2).SetDistance(200)

	// Verify pre-sort state
	if g.GetNodes() != 3 {
		t.Fatalf("expected 3 nodes, got %d", g.GetNodes())
	}
	if g.GetEdges() != 2 {
		t.Fatalf("expected 2 edges, got %d", g.GetEdges())
	}

	SortGraphAlongHilbertCurve(g)

	// After sort, node/edge counts should be preserved.
	if g.GetNodes() != 3 {
		t.Fatalf("expected 3 nodes after sort, got %d", g.GetNodes())
	}
	if g.GetEdges() != 2 {
		t.Fatalf("expected 2 edges after sort, got %d", g.GetEdges())
	}

	// Verify all coordinates still exist (possibly at different node IDs).
	lons := make(map[float64]bool)
	for node := range g.GetNodes() {
		lon := na.GetLon(node)
		lons[math.Round(lon)] = true
	}
	for _, expected := range []float64{-90, 0, 90} {
		if !lons[expected] {
			t.Errorf("expected lon %f to exist after sort", expected)
		}
	}
}

func TestSortGraphPreservesTopology(t *testing.T) {
	// Build a triangle graph and verify all edges still connect correct nodes after sort.
	g := NewBaseGraphBuilder(4).CreateGraph()
	na := g.GetNodeAccess()

	na.SetNode(0, 42.5, 1.5, 0)
	na.SetNode(1, 42.6, 1.6, 0)
	na.SetNode(2, 42.4, 1.4, 0)

	g.Edge(0, 1).SetDistance(100)
	g.Edge(1, 2).SetDistance(200)
	g.Edge(0, 2).SetDistance(150)

	// Collect all edges as (lat1, lon1) -> (lat2, lon2) pairs before sort.
	type edgePair struct {
		latA, lonA, latB, lonB float64
		dist                   float64
	}
	getBefore := func() []edgePair {
		var pairs []edgePair
		for e := range g.GetEdges() {
			ptr := g.Store.ToEdgePointer(e)
			nA := g.Store.GetNodeA(ptr)
			nB := g.Store.GetNodeB(ptr)
			pairs = append(pairs, edgePair{
				latA: na.GetLat(nA), lonA: na.GetLon(nA),
				latB: na.GetLat(nB), lonB: na.GetLon(nB),
				dist: g.GetDist(e),
			})
		}
		return pairs
	}

	before := getBefore()

	SortGraphAlongHilbertCurve(g)

	after := getBefore()

	// Same number of edges
	if len(before) != len(after) {
		t.Fatalf("edge count changed: %d -> %d", len(before), len(after))
	}

	// Each before-edge must be found in after-edges (order may differ, and nodeA/nodeB may swap).
	for _, be := range before {
		found := false
		for _, ae := range after {
			if math.Abs(ae.dist-be.dist) < 0.1 &&
				((closeEnough(ae.latA, be.latA) && closeEnough(ae.lonA, be.lonA) &&
					closeEnough(ae.latB, be.latB) && closeEnough(ae.lonB, be.lonB)) ||
					(closeEnough(ae.latA, be.latB) && closeEnough(ae.lonA, be.lonB) &&
						closeEnough(ae.latB, be.latA) && closeEnough(ae.lonB, be.lonA))) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("edge (%.4f,%.4f)-(%.4f,%.4f) dist=%.1f missing after sort",
				be.latA, be.lonA, be.latB, be.lonB, be.dist)
		}
	}
}

func TestSortGraphWithTurnCosts(t *testing.T) {
	g := NewBaseGraphBuilder(4).SetWithTurnCosts(true).CreateGraph()
	na := g.GetNodeAccess()

	na.SetNode(0, 42.5, 1.5, 0)
	na.SetNode(1, 42.6, 1.6, 0)
	na.SetNode(2, 42.4, 1.4, 0)

	g.Edge(0, 1).SetDistance(100)
	g.Edge(1, 2).SetDistance(200)

	// Add a turn cost entry at node 1 (from edge 0 to edge 1)
	idx := g.TurnCostStorage.FindOrCreateEntry(na, 0, 1, 1)
	g.TurnCostStorage.SetFlags(idx, 42)

	SortGraphAlongHilbertCurve(g)

	// After sort, turn cost count should be preserved.
	if g.TurnCostStorage.Count() != 1 {
		t.Fatalf("expected 1 turn cost entry after sort, got %d", g.TurnCostStorage.Count())
	}

	// Graph structure should still be intact.
	if g.GetNodes() != 3 {
		t.Fatalf("expected 3 nodes after sort, got %d", g.GetNodes())
	}
	if g.GetEdges() != 2 {
		t.Fatalf("expected 2 edges after sort, got %d", g.GetEdges())
	}
}

func TestSortGraphAdjacencyIntact(t *testing.T) {
	// Verify that edge exploration still works after sort.
	g := NewBaseGraphBuilder(4).CreateGraph()
	na := g.GetNodeAccess()

	na.SetNode(0, 10, 20, 0)
	na.SetNode(1, 30, 40, 0)
	na.SetNode(2, 50, 60, 0)
	na.SetNode(3, 70, 80, 0)

	g.Edge(0, 1).SetDistance(100)
	g.Edge(0, 2).SetDistance(200)
	g.Edge(1, 3).SetDistance(300)
	g.Edge(2, 3).SetDistance(400)

	SortGraphAlongHilbertCurve(g)

	// Every node should have the right degree.
	explorer := g.CreateEdgeExplorer(routingutil.AllEdges)
	totalDegree := 0
	for node := range g.GetNodes() {
		iter := explorer.SetBaseNode(node)
		for iter.Next() {
			totalDegree++
		}
	}
	// Each edge is counted from both endpoints, so total degree = 2 * edges.
	if totalDegree != 2*g.GetEdges() {
		t.Fatalf("expected total degree %d, got %d", 2*g.GetEdges(), totalDegree)
	}
}

func closeEnough(a, b float64) bool {
	return math.Abs(a-b) < 1e-4
}
