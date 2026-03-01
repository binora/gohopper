package index

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"gohopper/core/routing/ev"
	routeutil "gohopper/core/routing/util"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// --- test-local encoded values (mirror the Java field-level declarations) ---

func newSpeedEnc() ev.DecimalEncodedValue {
	return ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
}

func newAccessAndSpeedEncs() (ev.BooleanEncodedValue, ev.DecimalEncodedValue, int) {
	acc := ev.NewSimpleBooleanEncodedValueDir("access", true)
	spd := ev.NewDecimalEncodedValueImpl("speed", 5, 5, false)
	cfg := ev.NewInitializerConfig()
	acc.Init(cfg)
	spd.Init(cfg)
	return acc, spd, cfg.GetRequiredBytes()
}

func initEncs(evs ...ev.EncodedValue) int {
	cfg := ev.NewInitializerConfig()
	for _, e := range evs {
		e.Init(cfg)
	}
	return cfg.GetRequiredBytes()
}

// --- helper graph builders ---

// initSimpleGraph sets up the 7-node "simple graph".
//
//	 6 |        4
//	 5 |
//	   |     6
//	 4 |              5
//	 3 |
//	 2 |    1
//	 1 |          3
//	 0 |        2
//	-1 | 0
//	---|-------------------
//	   |-2 -1 0 1 2 3 4
func initSimpleGraph(g *storage.BaseGraph) {
	na := g.GetNodeAccess()
	na.SetNode(0, -1, -2, 0)
	na.SetNode(1, 2, -1, 0)
	na.SetNode(2, 0, 1, 0)
	na.SetNode(3, 1, 2, 0)
	na.SetNode(4, 6, 1, 0)
	na.SetNode(5, 4, 4, 0)
	na.SetNode(6, 4.5, -0.5, 0)
	g.Edge(0, 1)
	g.Edge(0, 2)
	g.Edge(2, 3)
	g.Edge(3, 4)
	g.Edge(1, 4)
	g.Edge(3, 5)
	// make sure 6 is connected
	g.Edge(6, 4)
}

// createTestGraph builds the 5-node test graph.
func createTestGraph(bytesForFlags int, speedEnc ev.DecimalEncodedValue) *storage.BaseGraph {
	graph := storage.NewBaseGraphBuilder(bytesForFlags).CreateGraph()
	na := graph.GetNodeAccess()
	na.SetNode(0, 0.5, -0.5, 0)
	na.SetNode(1, -0.5, -0.5, 0)
	na.SetNode(2, -1, -1, 0)
	na.SetNode(3, -0.4, 0.9, 0)
	na.SetNode(4, -0.6, 1.6, 0)
	graph.Edge(0, 1).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(0, 2).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(0, 4).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(1, 3).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(2, 3).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(2, 4).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(3, 4).SetDecimalBothDir(speedEnc, 60, 60)
	return graph
}

// createTestGraphWithWayGeometry builds the test graph with pillar nodes A/B.
func createTestGraphWithWayGeometry(bytesForFlags int, speedEnc ev.DecimalEncodedValue) *storage.BaseGraph {
	graph := storage.NewBaseGraphBuilder(bytesForFlags).CreateGraph()
	na := graph.GetNodeAccess()
	na.SetNode(0, 0.5, -0.5, 0)
	na.SetNode(1, -0.5, -0.5, 0)
	na.SetNode(2, -1, -1, 0)
	na.SetNode(3, -0.4, 0.9, 0)
	na.SetNode(4, -0.6, 1.6, 0)
	graph.Edge(0, 1).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(0, 2).SetDecimalBothDir(speedEnc, 60, 60)
	// insert A and B, without this we would get 0 for 0,0
	graph.Edge(0, 4).SetDecimalBothDir(speedEnc, 60, 60).SetWayGeometry(util.CreatePointList(1, 1))
	graph.Edge(1, 3).SetDecimalBothDir(speedEnc, 60, 60).SetWayGeometry(util.CreatePointList(0, 0))
	graph.Edge(2, 3).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(2, 4).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(3, 4).SetDecimalBothDir(speedEnc, 60, 60)
	return graph
}

// createTestGraph2 builds the large 35-node test graph.
func createTestGraph2(bytesForFlags int, speedEnc ev.DecimalEncodedValue) *storage.BaseGraph {
	graph := storage.NewBaseGraphBuilder(bytesForFlags).CreateGraph()
	na := graph.GetNodeAccess()

	na.SetNode(0, 49.94653, 11.57114, 0)
	na.SetNode(1, 49.94653, 11.57214, 0)
	na.SetNode(2, 49.94653, 11.57314, 0)
	na.SetNode(3, 49.94653, 11.57414, 0)
	na.SetNode(4, 49.94653, 11.57514, 0)
	na.SetNode(5, 49.94653, 11.57614, 0)
	na.SetNode(6, 49.94653, 11.57714, 0)
	na.SetNode(7, 49.94653, 11.57814, 0)

	na.SetNode(8, 49.94553, 11.57214, 0)
	na.SetNode(9, 49.94553, 11.57314, 0)
	na.SetNode(10, 49.94553, 11.57414, 0)
	na.SetNode(11, 49.94553, 11.57514, 0)
	na.SetNode(12, 49.94553, 11.57614, 0)
	na.SetNode(13, 49.94553, 11.57714, 0)

	na.SetNode(14, 49.94753, 11.57214, 0)
	na.SetNode(15, 49.94753, 11.57314, 0)
	na.SetNode(16, 49.94753, 11.57614, 0)
	na.SetNode(17, 49.94753, 11.57814, 0)

	na.SetNode(18, 49.94853, 11.57114, 0)
	na.SetNode(19, 49.94853, 11.57214, 0)
	na.SetNode(20, 49.94853, 11.57814, 0)

	na.SetNode(21, 49.94953, 11.57214, 0)
	na.SetNode(22, 49.94953, 11.57614, 0)

	na.SetNode(23, 49.95053, 11.57114, 0)
	na.SetNode(24, 49.95053, 11.57214, 0)
	na.SetNode(25, 49.95053, 11.57314, 0)
	na.SetNode(26, 49.95053, 11.57514, 0)
	na.SetNode(27, 49.95053, 11.57614, 0)
	na.SetNode(28, 49.95053, 11.57714, 0)
	na.SetNode(29, 49.95053, 11.57814, 0)

	na.SetNode(30, 49.95153, 11.57214, 0)
	na.SetNode(31, 49.95153, 11.57314, 0)
	na.SetNode(32, 49.95153, 11.57514, 0)
	na.SetNode(33, 49.95153, 11.57614, 0)
	na.SetNode(34, 49.95153, 11.57714, 0)

	// to create correct bounds — bottom left
	na.SetNode(100, 49.941, 11.56614, 0)
	// top right
	na.SetNode(101, 49.96053, 11.58814, 0)

	graph.Edge(0, 1).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(1, 2).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(2, 3).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(3, 4).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(4, 5).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(6, 7).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(2, 8).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(2, 9).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(3, 10).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(4, 11).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(5, 12).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(6, 13).SetDecimalBothDir(speedEnc, 60, 60)

	graph.Edge(1, 14).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(2, 15).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(5, 16).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(14, 15).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(16, 17).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(16, 20).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(16, 25).SetDecimalBothDir(speedEnc, 60, 60)

	graph.Edge(18, 14).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(18, 19).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(18, 21).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(19, 21).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(21, 24).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(23, 24).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(24, 25).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(26, 27).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(27, 28).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(28, 29).SetDecimalBothDir(speedEnc, 60, 60)

	graph.Edge(24, 30).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(24, 31).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(26, 32).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(26, 22).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(27, 33).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(28, 34).SetDecimalBothDir(speedEnc, 60, 60)
	return graph
}

// createSampleGraph builds the 17-node sample graph.
func createSampleGraph(bytesForFlags int, speedEnc ev.DecimalEncodedValue) *storage.BaseGraph {
	graph := storage.NewBaseGraphBuilder(bytesForFlags).CreateGraph()
	na := graph.GetNodeAccess()
	na.SetNode(0, 0, 1.0001, 0)
	na.SetNode(1, 1, 2, 0)
	na.SetNode(2, 0.5, 4.5, 0)
	na.SetNode(3, 1.5, 3.8, 0)
	na.SetNode(4, 2.01, 0.5, 0)
	na.SetNode(5, 2, 3, 0)
	na.SetNode(6, 3, 1.5, 0)
	na.SetNode(7, 2.99, 3.01, 0)
	na.SetNode(8, 3, 4, 0)
	na.SetNode(9, 3.3, 2.2, 0)
	na.SetNode(10, 4, 1, 0)
	na.SetNode(11, 4.1, 3, 0)
	na.SetNode(12, 4, 4.5, 0)
	na.SetNode(13, 4.5, 4.1, 0)
	na.SetNode(14, 5, 0, 0)
	na.SetNode(15, 4.9, 2.5, 0)
	na.SetNode(16, 5, 5, 0)

	graph.Edge(0, 1).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(2, 1).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(2, 3).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(5, 1).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(4, 5).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(12, 3).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(4, 10).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(5, 3).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(5, 8).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(5, 9).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(10, 6).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(9, 11).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(8, 11).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(8, 7).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(10, 13).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(10, 14).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(11, 15).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(12, 15).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(16, 15).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(16, 12).SetDecimalBothDir(speedEnc, 60, 60)
	return graph
}

// --- index helpers ---

func createIndexNoPrepare(g storage.Graph, resolution int) *LocationIndexTree {
	dir := storage.NewRAMDirectory("", false)
	tmpIDX := NewLocationIndexTree(g, dir)
	tmpIDX.SetResolution(resolution)
	return tmpIDX
}

func findClosestNode(idx LocationIndex, lat, lon float64) int {
	snap := idx.FindClosest(lat, lon, routeutil.AllEdges)
	if snap.GetSnappedPosition() != Tower {
		panic(fmt.Sprintf("expected TOWER position, got %v for (%v, %v)", snap.GetSnappedPosition(), lat, lon))
	}
	return snap.GetClosestNode()
}

func findClosestEdge(idx LocationIndex, lat, lon float64) int {
	return idx.FindClosest(lat, lon, routeutil.AllEdges).GetClosestEdge().GetEdge()
}

// --- edgeCollector implements Visitor to collect edge IDs ---

type edgeCollector struct {
	edges map[int]struct{}
}

func newEdgeCollector() *edgeCollector {
	return &edgeCollector{edges: make(map[int]struct{})}
}

func (c *edgeCollector) OnEdge(edgeID int) {
	c.edges[edgeID] = struct{}{}
}

func (c *edgeCollector) IsTileInfo() bool          { return false }
func (c *edgeCollector) OnTile(_ util.BBox, _ int) {}

func (c *edgeCollector) size() int {
	return len(c.edges)
}

// accessFilterAllEdges returns an EdgeFilter that accepts only edges where the
// given BooleanEncodedValue is true in at least one direction.
func accessFilterAllEdges(enc ev.BooleanEncodedValue) routeutil.EdgeFilter {
	return func(edge util.EdgeIteratorState) bool {
		return edge.GetBool(enc) || edge.GetReverseBool(enc)
	}
}

// getEdge finds the edge between base and adj in the given graph.
func getEdge(g *storage.BaseGraph, base, adj int) util.EdgeIteratorState {
	explorer := g.CreateEdgeExplorer(routeutil.AllEdges)
	iter := explorer.SetBaseNode(base)
	for iter.Next() {
		if iter.GetAdjNode() == adj {
			return iter
		}
	}
	return nil
}

// --- Tests ---

func TestSnappedPointAndGeometry(t *testing.T) {
	speedEnc := newSpeedEnc()
	bytesForFlags := initEncs(speedEnc)
	graph := createTestGraph(bytesForFlags, speedEnc)
	defer graph.Close()

	idx := createIndexNoPrepare(graph, 500000).PrepareIndex()
	defer idx.Close()

	// query directly the tower node
	res := idx.FindClosest(-0.4, 0.9, routeutil.AllEdges)
	if !res.IsValid() {
		t.Fatal("expected valid snap")
	}
	sp := res.GetSnappedPoint()
	if math.Abs(sp.Lat-(-0.4)) > 1e-6 || math.Abs(sp.Lon-0.9) > 1e-6 {
		t.Fatalf("expected snapped point (-0.4, 0.9), got (%v, %v)", sp.Lat, sp.Lon)
	}

	res = idx.FindClosest(-0.6, 1.6, routeutil.AllEdges)
	if !res.IsValid() {
		t.Fatal("expected valid snap")
	}
	sp = res.GetSnappedPoint()
	if math.Abs(sp.Lat-(-0.6)) > 1e-6 || math.Abs(sp.Lon-1.6) > 1e-6 {
		t.Fatalf("expected snapped point (-0.6, 1.6), got (%v, %v)", sp.Lat, sp.Lon)
	}

	// query the edge (1,3). The edge (0,4) has 27674 as distance
	res = idx.FindClosest(-0.2, 0.3, routeutil.AllEdges)
	if !res.IsValid() {
		t.Fatal("expected valid snap")
	}
	if math.Abs(res.GetQueryDistance()-26936) > 1 {
		t.Fatalf("expected query distance ~26936, got %v", res.GetQueryDistance())
	}
	sp = res.GetSnappedPoint()
	if math.Abs(sp.Lat-(-0.441624)) > 1e-3 || math.Abs(sp.Lon-0.317259) > 1e-3 {
		t.Fatalf("expected snapped point (-0.441624, 0.317259), got (%v, %v)", sp.Lat, sp.Lon)
	}
}

func TestBoundingBoxQuery2(t *testing.T) {
	speedEnc := newSpeedEnc()
	bytesForFlags := initEncs(speedEnc)
	graph := createTestGraph2(bytesForFlags, speedEnc)
	defer graph.Close()

	idx := createIndexNoPrepare(graph, 500).PrepareIndex().(*LocationIndexTree)
	defer idx.Close()

	collector := newEdgeCollector()
	QueryBBox(idx, graph.GetBounds(), collector)
	if collector.size() != graph.GetEdges() {
		t.Fatalf("expected %d edges, got %d", graph.GetEdges(), collector.size())
	}
}

func TestBoundingBoxQuery1(t *testing.T) {
	speedEnc := newSpeedEnc()
	bytesForFlags := initEncs(speedEnc)
	graph := createTestGraph2(bytesForFlags, speedEnc)
	defer graph.Close()

	idx := createIndexNoPrepare(graph, 500).PrepareIndex().(*LocationIndexTree)
	defer idx.Close()

	collector := newEdgeCollector()
	bbox := util.NewBBox(11.57114, 11.57814, 49.94553, 49.94853)
	QueryBBox(idx, bbox, collector)
	if collector.size() != graph.GetEdges() {
		t.Fatalf("expected %d edges, got %d", graph.GetEdges(), collector.size())
	}
}

func TestMoreReal(t *testing.T) {
	speedEnc := newSpeedEnc()
	bytesForFlags := initEncs(speedEnc)
	graph := storage.NewBaseGraphBuilder(bytesForFlags).CreateGraph()
	defer graph.Close()

	na := graph.GetNodeAccess()
	na.SetNode(1, 51.2492152, 9.4317166, 0)
	na.SetNode(0, 52, 9, 0)
	na.SetNode(2, 51.2, 9.4, 0)
	na.SetNode(3, 49, 10, 0)

	graph.Edge(1, 0).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(0, 2).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(0, 3).SetDecimalBothDir(speedEnc, 60, 60).SetWayGeometry(util.CreatePointList(51.21, 9.43))

	idx := createIndexNoPrepare(graph, 500000).PrepareIndex()
	defer idx.Close()

	if got := findClosestEdge(idx, 51.2, 9.4); got != 1 {
		t.Fatalf("expected edge 1, got %d", got)
	}
}

func TestWayGeometry(t *testing.T) {
	speedEnc := newSpeedEnc()
	bytesForFlags := initEncs(speedEnc)
	g := createTestGraphWithWayGeometry(bytesForFlags, speedEnc)
	defer g.Close()

	idx := createIndexNoPrepare(g, 500000).PrepareIndex()
	defer idx.Close()

	if got := findClosestEdge(idx, 0, 0); got != 3 {
		t.Fatalf("expected edge 3, got %d", got)
	}
	if got := findClosestEdge(idx, 0, 0.1); got != 3 {
		t.Fatalf("expected edge 3, got %d", got)
	}
	if got := findClosestEdge(idx, 0.1, 0.1); got != 3 {
		t.Fatalf("expected edge 3, got %d", got)
	}
	if got := findClosestNode(idx, -0.5, -0.5); got != 1 {
		t.Fatalf("expected node 1, got %d", got)
	}
}

func TestFindingWayGeometry(t *testing.T) {
	speedEnc := newSpeedEnc()
	bytesForFlags := initEncs(speedEnc)
	g := storage.NewBaseGraphBuilder(bytesForFlags).CreateGraph()
	defer g.Close()

	na := g.GetNodeAccess()
	na.SetNode(10, 51.2492152, 9.4317166, 0)
	na.SetNode(20, 52, 9, 0)
	na.SetNode(30, 51.2, 9.4, 0)
	na.SetNode(50, 49, 10, 0)
	g.Edge(20, 50).SetDecimalBothDir(speedEnc, 60, 60).SetWayGeometry(util.CreatePointList(51.25, 9.43))
	g.Edge(10, 20).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(20, 30).SetDecimalBothDir(speedEnc, 60, 60)

	idx := createIndexNoPrepare(g, 2000).PrepareIndex()
	defer idx.Close()

	if got := findClosestEdge(idx, 51.25, 9.43); got != 0 {
		t.Fatalf("expected edge 0, got %d", got)
	}
}

func TestEdgeFilter(t *testing.T) {
	speedEnc := newSpeedEnc()
	bytesForFlags := initEncs(speedEnc)
	graph := createTestGraph(bytesForFlags, speedEnc)
	defer graph.Close()

	idx := createIndexNoPrepare(graph, 500000).PrepareIndex().(*LocationIndexTree)
	defer idx.Close()

	snap := idx.FindClosest(-0.6, -0.6, routeutil.AllEdges)
	if snap.GetClosestNode() != 1 {
		t.Fatalf("expected node 1, got %d", snap.GetClosestNode())
	}

	snap = idx.FindClosest(-0.6, -0.6, func(edge util.EdgeIteratorState) bool {
		return edge.GetBaseNode() == 2 || edge.GetAdjNode() == 2
	})
	if snap.GetClosestNode() != 2 {
		t.Fatalf("expected node 2, got %d", snap.GetClosestNode())
	}
}

func TestRMin(t *testing.T) {
	speedEnc := newSpeedEnc()
	bytesForFlags := initEncs(speedEnc)
	graph := createTestGraph(bytesForFlags, speedEnc)
	defer graph.Close()

	idx := createIndexNoPrepare(graph, 50000).PrepareIndex().(*LocationIndexTree)
	defer idx.Close()

	rmin2 := idx.CalculateRMin(0.05, -0.3, 1)
	check2 := util.DistPlane.CalcDist(0.05, math.Abs(graph.GetNodeAccess().GetLat(0)), -0.3, -0.3)
	if rmin2-check2 >= 0.0001 {
		t.Fatalf("expected rmin2-check2 < 0.0001, got %v", rmin2-check2)
	}
}

func TestSearchWithFilter_issue318(t *testing.T) {
	carAccessEnc := ev.NewSimpleBooleanEncodedValueDir("car_access", true)
	carSpeedEnc := ev.NewDecimalEncodedValueImpl("car_speed", 5, 5, false)
	bikeAccessEnc := ev.NewSimpleBooleanEncodedValueDir("bike_access", true)
	bikeSpeedEnc := ev.NewDecimalEncodedValueImpl("bike_speed", 4, 2, false)

	bytesForFlags := initEncs(carAccessEnc, carSpeedEnc, bikeAccessEnc, bikeSpeedEnc)
	graph := storage.NewBaseGraphBuilder(bytesForFlags).CreateGraph()
	defer graph.Close()

	na := graph.GetNodeAccess()

	// distance from point to point is roughly 1 km
	maxN := 5
	for latIdx := 0; latIdx < maxN; latIdx++ {
		for lonIdx := 0; lonIdx < maxN; lonIdx++ {
			index := lonIdx*10 + latIdx
			na.SetNode(index, 0.01*float64(latIdx), 0.01*float64(lonIdx), 0)
			if latIdx < maxN-1 {
				util.SetSpeed(60, true, true, carAccessEnc, carSpeedEnc, graph.Edge(index, index+1))
			}
			if lonIdx < maxN-1 {
				util.SetSpeed(60, true, true, carAccessEnc, carSpeedEnc, graph.Edge(index, index+10))
			}
		}
	}

	// reduce access for bike to two edges only
	iter := graph.GetAllEdges()
	for iter.Next() {
		iter.SetBoolBothDir(bikeAccessEnc, false, false)
	}
	edge01 := getEdge(graph, 0, 1)
	edge12 := getEdge(graph, 1, 2)
	if edge01 != nil {
		edge01.SetBoolBothDir(bikeAccessEnc, true, true)
	}
	if edge12 != nil {
		edge12.SetBoolBothDir(bikeAccessEnc, true, true)
	}

	idx := createIndexNoPrepare(graph, 500)
	idx.PrepareIndex()
	idx.SetMaxRegionSearch(8)
	defer idx.Close()

	carFilter := accessFilterAllEdges(carAccessEnc)
	snap := idx.FindClosest(0.03, 0.03, carFilter)
	if !snap.IsValid() {
		t.Fatal("expected valid snap for car")
	}
	if snap.GetClosestNode() != 33 {
		t.Fatalf("expected node 33, got %d", snap.GetClosestNode())
	}

	bikeFilter := accessFilterAllEdges(bikeAccessEnc)
	snap = idx.FindClosest(0.03, 0.03, bikeFilter)
	if !snap.IsValid() {
		t.Fatal("expected valid snap for bike")
	}
	if snap.GetClosestNode() != 2 {
		t.Fatalf("expected node 2, got %d", snap.GetClosestNode())
	}
}

func TestCrossBoundaryNetwork_issue667(t *testing.T) {
	speedEnc := newSpeedEnc()
	bytesForFlags := initEncs(speedEnc)
	graph := storage.NewBaseGraphBuilder(bytesForFlags).CreateGraph()
	defer graph.Close()

	na := graph.GetNodeAccess()
	na.SetNode(0, 0.1, 179.5, 0)
	na.SetNode(1, 0.1, 179.9, 0)
	na.SetNode(2, 0.1, -179.8, 0)
	na.SetNode(3, 0.1, -179.5, 0)
	na.SetNode(4, 0, 179.5, 0)
	na.SetNode(5, 0, 179.9, 0)
	na.SetNode(6, 0, -179.8, 0)
	na.SetNode(7, 0, -179.5, 0)

	graph.Edge(0, 1).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(0, 4).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(1, 5).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(4, 5).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(2, 3).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(2, 6).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(3, 7).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Edge(6, 7).SetDecimalBothDir(speedEnc, 60, 60)

	// cross boundary edges
	graph.Edge(1, 2).SetDecimalBothDir(speedEnc, 60, 60).SetWayGeometry(util.CreatePointList(0, 180, 0, -180))
	graph.Edge(5, 6).SetDecimalBothDir(speedEnc, 60, 60)

	idx := createIndexNoPrepare(graph, 500)
	idx.PrepareIndex()
	defer idx.Close()

	if graph.GetNodes() <= 0 {
		t.Fatal("expected nodes > 0")
	}
	for i := 0; i < graph.GetNodes(); i++ {
		snap := idx.FindClosest(na.GetLat(i), na.GetLon(i), routeutil.AllEdges)
		if snap.GetClosestNode() != i {
			t.Fatalf("expected closest node %d, got %d", i, snap.GetClosestNode())
		}
	}
}

func TestSimpleGraph(t *testing.T) {
	accessEnc, speedEnc, bytesForFlags := newAccessAndSpeedEncs()
	g := storage.NewBaseGraphBuilder(bytesForFlags).CreateGraph()
	defer g.Close()

	initSimpleGraph(g)
	edge := g.GetAllEdges()
	for edge.Next() {
		util.SetSpeeds(60, 60, accessEnc, speedEnc, edge)
	}

	idx := createIndexNoPrepare(g, 500000).PrepareIndex().(*LocationIndexTree)
	defer idx.Close()

	if got := findClosestEdge(idx, 5, 2); got != 3 {
		t.Fatalf("expected edge 3, got %d", got)
	}
	if got := findClosestEdge(idx, 1.5, 2); got != 3 {
		t.Fatalf("expected edge 3, got %d", got)
	}
	if got := findClosestEdge(idx, -1, -1); got != 1 {
		t.Fatalf("expected edge 1, got %d", got)
	}
	if got := findClosestEdge(idx, 4, 0); got != 4 {
		t.Fatalf("expected edge 4, got %d", got)
	}
}

func TestSimpleGraph2(t *testing.T) {
	accessEnc, speedEnc, bytesForFlags := newAccessAndSpeedEncs()
	g := storage.NewBaseGraphBuilder(bytesForFlags).CreateGraph()
	defer g.Close()

	initSimpleGraph(g)
	edge := g.GetAllEdges()
	for edge.Next() {
		util.SetSpeeds(60, 60, accessEnc, speedEnc, edge)
	}

	idx := createIndexNoPrepare(g, 500000).PrepareIndex().(*LocationIndexTree)
	defer idx.Close()

	if got := findClosestEdge(idx, 5, 2); got != 3 {
		t.Fatalf("expected edge 3, got %d", got)
	}
	if got := findClosestEdge(idx, 1.5, 2); got != 3 {
		t.Fatalf("expected edge 3, got %d", got)
	}
	if got := findClosestEdge(idx, -1, -1); got != 1 {
		t.Fatalf("expected edge 1, got %d", got)
	}
	if got := findClosestNode(idx, 4.5, -0.5); got != 6 {
		t.Fatalf("expected node 6, got %d", got)
	}
	if got := findClosestEdge(idx, 4, 1); got != 3 {
		t.Fatalf("expected edge 3, got %d", got)
	}
	if got := findClosestEdge(idx, 4, 0); got != 4 {
		t.Fatalf("expected edge 4, got %d", got)
	}
	if got := findClosestNode(idx, 4, -2); got != 6 {
		t.Fatalf("expected node 6, got %d", got)
	}
	if got := findClosestEdge(idx, 3, 3); got != 5 {
		t.Fatalf("expected edge 5, got %d", got)
	}
}

func TestSinglePoints120(t *testing.T) {
	speedEnc := newSpeedEnc()
	bytesForFlags := initEncs(speedEnc)
	g := createSampleGraph(bytesForFlags, speedEnc)
	defer g.Close()

	idx := createIndexNoPrepare(g, 500000).PrepareIndex().(*LocationIndexTree)
	defer idx.Close()

	if got := findClosestEdge(idx, 1.637, 2.23); got != 3 {
		t.Fatalf("expected edge 3, got %d", got)
	}
	if got := findClosestEdge(idx, 3.649, 1.375); got != 10 {
		t.Fatalf("expected edge 10, got %d", got)
	}
	if got := findClosestNode(idx, 3.3, 2.2); got != 9 {
		t.Fatalf("expected node 9, got %d", got)
	}
	if got := findClosestNode(idx, 3.0, 1.5); got != 6 {
		t.Fatalf("expected node 6, got %d", got)
	}
	if got := findClosestEdge(idx, 3.8, 0); got != 15 {
		t.Fatalf("expected edge 15, got %d", got)
	}
	if got := findClosestEdge(idx, 3.8466, 0.021); got != 15 {
		t.Fatalf("expected edge 15, got %d", got)
	}
}

func TestSinglePoints32(t *testing.T) {
	speedEnc := newSpeedEnc()
	bytesForFlags := initEncs(speedEnc)
	g := createSampleGraph(bytesForFlags, speedEnc)
	defer g.Close()

	idx := createIndexNoPrepare(g, 500000).PrepareIndex().(*LocationIndexTree)
	defer idx.Close()

	if got := findClosestEdge(idx, 3.649, 1.375); got != 10 {
		t.Fatalf("expected edge 10, got %d", got)
	}
	if got := findClosestEdge(idx, 3.8465748, 0.021762699); got != 15 {
		t.Fatalf("expected edge 15, got %d", got)
	}
	if got := findClosestEdge(idx, 2.485, 1.373); got != 4 {
		t.Fatalf("expected edge 4, got %d", got)
	}
	if got := findClosestEdge(idx, 0.64628404, 0.53006625); got != 0 {
		t.Fatalf("expected edge 0, got %d", got)
	}
}

func TestNoErrorOnEdgeCase_lastIndex(t *testing.T) {
	// Empty encoding manager — just use minimal bytesForFlags
	bytesForFlags := 4
	locs := 10000
	g := storage.NewBaseGraphBuilder(bytesForFlags).CreateGraph()
	defer g.Close()

	na := g.GetNodeAccess()
	r := rand.New(rand.NewSource(12))
	for i := 0; i < locs; i++ {
		na.SetNode(i, float64(float32(r.Float64()))*10+10, float64(float32(r.Float64()))*10+10, 0)
	}
	idx := createIndexNoPrepare(g, 200).PrepareIndex()
	idx.Close()
}

func TestDifferentVehicles(t *testing.T) {
	carAccessEnc := ev.NewSimpleBooleanEncodedValueDir("car_access", true)
	carSpeedEnc := ev.NewDecimalEncodedValueImpl("car_speed", 5, 5, false)
	footAccessEnc := ev.NewSimpleBooleanEncodedValueDir("foot_access", true)
	footSpeedEnc := ev.NewDecimalEncodedValueImpl("foot_speed", 4, 1, false)

	bytesForFlags := initEncs(carAccessEnc, carSpeedEnc, footAccessEnc, footSpeedEnc)
	g := storage.NewBaseGraphBuilder(bytesForFlags).CreateGraph()
	defer g.Close()

	initSimpleGraph(g)
	edge := g.GetAllEdges()
	for edge.Next() {
		util.SetSpeeds(60, 60, carAccessEnc, carSpeedEnc, edge)
		util.SetSpeeds(10, 10, footAccessEnc, footSpeedEnc, edge)
	}

	idx := createIndexNoPrepare(g, 500000).PrepareIndex().(*LocationIndexTree)
	defer idx.Close()

	if got := findClosestEdge(idx, 1, -1); got != 0 {
		t.Fatalf("expected edge 0, got %d", got)
	}

	// now make all edges from node 1 accessible for CAR only
	explorer := g.CreateEdgeExplorer(routeutil.AllEdges)
	iter := explorer.SetBaseNode(1)
	for iter.Next() {
		iter.SetBoolBothDir(footAccessEnc, false, false)
	}

	idx2 := createIndexNoPrepare(g, 500000).PrepareIndex().(*LocationIndexTree)
	defer idx2.Close()

	footFilter := accessFilterAllEdges(footAccessEnc)
	snap := idx2.FindClosest(1, -1, footFilter)
	if snap.GetClosestNode() != 2 {
		t.Fatalf("expected node 2, got %d", snap.GetClosestNode())
	}
}

func TestCloseToTowerNode(t *testing.T) {
	for _, snapAtBase := range []bool{true, false} {
		name := "snapAtBase"
		if !snapAtBase {
			name = "snapAtAdj"
		}
		t.Run(name, func(t *testing.T) {
			speedEnc := newSpeedEnc()
			bytesForFlags := initEncs(speedEnc)
			graph := storage.NewBaseGraphBuilder(bytesForFlags).CreateGraph()
			defer graph.Close()

			na := graph.GetNodeAccess()
			na.SetNode(0, 51.985500, 19.254000, 0)
			na.SetNode(1, 51.986000, 19.255000, 0)

			snapNode := 0
			base := 0
			adj := 1
			if !snapAtBase {
				base = 1
				adj = 0
			}
			dist := util.DistPlane.CalcDist(na.GetLat(0), na.GetLon(0), na.GetLat(1), na.GetLon(1))
			graph.Edge(base, adj).SetDistance(dist)

			idx := NewLocationIndexTree(graph, storage.NewRAMDirectory("", false))
			idx.PrepareIndex()
			defer idx.Close()

			queryLat := 51.9855003
			queryLon := 19.2540003
			distFromTower := util.DistPlane.CalcDist(queryLat, queryLon, na.GetLat(snapNode), na.GetLon(snapNode))
			if distFromTower >= 0.1 {
				t.Fatalf("expected dist from tower < 0.1, got %v", distFromTower)
			}
			snap := idx.FindClosest(queryLat, queryLon, routeutil.AllEdges)
			if snap.GetSnappedPosition() != Tower {
				t.Fatalf("expected TOWER snap, got %v", snap.GetSnappedPosition())
			}
		})
	}
}

func TestQueryBehindBeforeOrBehindLastTowerNode(t *testing.T) {
	speedEnc := newSpeedEnc()
	bytesForFlags := initEncs(speedEnc)
	graph := storage.NewBaseGraphBuilder(bytesForFlags).CreateGraph()
	defer graph.Close()

	na := graph.GetNodeAccess()
	na.SetNode(0, 51.985000, 19.254000, 0)
	na.SetNode(1, 51.986000, 19.255000, 0)

	dist := util.DistPlane.CalcDist(na.GetLat(0), na.GetLon(0), na.GetLat(1), na.GetLon(1))
	edge := graph.Edge(0, 1).SetDistance(dist)
	edge.SetWayGeometry(util.CreatePointList(51.985500, 19.254500))

	idx := NewLocationIndexTree(graph, storage.NewRAMDirectory("", false))
	idx.PrepareIndex()
	defer idx.Close()

	{
		// snap before last tower node
		var output []string
		idx.TraverseEdge(51.985700, 19.254700, edge, func(node int, normedDist float64, wayIndex int, pos Position) {
			rounded := int(math.Round(util.DistPlane.CalcDenormalizedDist(normedDist)))
			output = append(output, fmt.Sprintf("%d, %d, %d, %s", node, rounded, wayIndex, pos))
		})
		expected := []string{
			"1, 39, 2, TOWER",
			"1, 26, 1, PILLAR",
			"1, 0, 1, EDGE",
		}
		if len(output) != len(expected) {
			t.Fatalf("expected %d entries, got %d: %v", len(expected), len(output), output)
		}
		for i := range expected {
			if output[i] != expected[i] {
				t.Fatalf("entry %d: expected %q, got %q", i, expected[i], output[i])
			}
		}
	}

	{
		// snap behind last tower node
		var output []string
		idx.TraverseEdge(51.986100, 19.255100, edge, func(node int, normedDist float64, wayIndex int, pos Position) {
			rounded := int(math.Round(util.DistPlane.CalcDenormalizedDist(normedDist)))
			output = append(output, fmt.Sprintf("%d, %d, %d, %s", node, rounded, wayIndex, pos))
		})
		expected := []string{
			"1, 13, 2, TOWER",
			"1, 78, 1, PILLAR",
		}
		if len(output) != len(expected) {
			t.Fatalf("expected %d entries, got %d: %v", len(expected), len(output), output)
		}
		for i := range expected {
			if output[i] != expected[i] {
				t.Fatalf("entry %d: expected %q, got %q", i, expected[i], output[i])
			}
		}
	}
}
