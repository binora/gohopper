package osm

import (
	"testing"

	"gohopper/core/reader"
	"gohopper/core/storage"
	"gohopper/core/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testNodeAccess is an in-memory NodeAccess for tests.
type testNodeAccess struct {
	lats []float64
	lons []float64
	eles []float64
	is3D bool
}

func newTestNodeAccess(is3D bool) *testNodeAccess {
	return &testNodeAccess{is3D: is3D}
}

func (na *testNodeAccess) Is3D() bool    { return na.is3D }
func (na *testNodeAccess) Dimension() int { if na.is3D { return 3 }; return 2 }

func (na *testNodeAccess) EnsureNode(nodeID int) {
	for len(na.lats) <= nodeID {
		na.lats = append(na.lats, 0)
		na.lons = append(na.lons, 0)
		na.eles = append(na.eles, 0)
	}
}

func (na *testNodeAccess) SetNode(nodeID int, lat, lon, ele float64) {
	na.EnsureNode(nodeID)
	na.lats[nodeID] = lat
	na.lons[nodeID] = lon
	na.eles[nodeID] = ele
}

func (na *testNodeAccess) GetLat(nodeID int) float64 { return na.lats[nodeID] }
func (na *testNodeAccess) GetLon(nodeID int) float64 { return na.lons[nodeID] }
func (na *testNodeAccess) GetEle(nodeID int) float64 { return na.eles[nodeID] }
func (na *testNodeAccess) GetTurnCostIndex(nodeID int) int { return 0 }
func (na *testNodeAccess) SetTurnCostIndex(nodeID int, value int) {}

// edgeRecord captures an edge created by the WaySegmentParser.
type edgeRecord struct {
	from, to int
	points   int
	wayID    int64
	barrier  bool
}

func TestWaySegmentParserBasic(t *testing.T) {
	// Test with test-osm.xml:
	// way 10: nodes 10→20→30 (highway=motorway_link)
	// way 11: nodes 20→40→50 (highway=service)
	// Node 20 is shared → junction
	na := newTestNodeAccess(false)
	dir := storage.NewRAMDirectory("", true)

	var edges []edgeRecord
	parser := NewWaySegmentParserBuilder(na, dir).
		SetWayFilter(func(way *reader.ReaderWay) bool {
			return way.GetTag("highway") != ""
		}).
		SetEdgeHandler(func(from, to int, pl *util.PointList, way *reader.ReaderWay, nodeTags []map[string]any) {
			edges = append(edges, edgeRecord{
				from:    from,
				to:      to,
				points:  pl.Size(),
				wayID:   way.GetID(),
				barrier: way.GetTag("gh:barrier_edge") != "",
			})
		}).
		Build()

	err := parser.ReadOSM("testdata/test-osm.xml")
	require.NoError(t, err)

	// Way 10 (10→20→30): node 20 is junction (shared with way 11), so splits into:
	//   edge: 10→20 (2 points)
	//   edge: 20→30 (2 points)
	// Way 11 (20→40→50): node 20 is junction, so splits into:
	//   edge: 20→50 (3 points, since 40 is pillar)
	// Wait — node 20 is already a tower (junction from pass1). Node 40 is intermediate (pillar).
	// Way 11 has tower at 20, pillar at 40, tower at 50 (end). So way 11 = one segment: 20→40→50.
	// Actually: way 10 nodes [10, 20, 30]. Node 10 = END, 20 = JUNCTION (both ways share it), 30 = END→CONNECTION? No.
	// Pass1 for way 10: node 10 END, node 20 INTERMEDIATE, node 30 END
	// Pass1 for way 11: node 20 END (prev=INTERMEDIATE → not END, so JUNCTION), node 40 INTERMEDIATE, node 50 END
	// So: 10=END→tower, 20=JUNCTION→tower, 30=END→tower (only appears at end of way 10)
	// 40=INTERMEDIATE→pillar, 50=END→tower (only appears at end of way 11)
	// Way 10 splits at junction 20: [10→20], [20→30] → 2 edges
	// Way 11: [20→40→50] → 1 edge (40 is pillar between two towers)
	assert.Equal(t, 3, len(edges), "expected 3 edges")

	// Verify edges are from way 10 and way 11
	wayEdgeCount := map[int64]int{}
	for _, e := range edges {
		wayEdgeCount[e.wayID]++
	}
	assert.Equal(t, 2, wayEdgeCount[10], "way 10 should produce 2 edges")
	assert.Equal(t, 1, wayEdgeCount[11], "way 11 should produce 1 edge")

	// The way 11 edge should have 3 points (20→40→50)
	for _, e := range edges {
		if e.wayID == 11 {
			assert.Equal(t, 3, e.points, "way 11 edge should have 3 points including pillar")
		}
	}
}

func TestWaySegmentParserBarrier(t *testing.T) {
	// Create a simple way with a barrier node in the middle.
	// Way: nodes [1, 2, 3] where node 2 is a barrier.
	na := newTestNodeAccess(false)
	dir := storage.NewRAMDirectory("", true)

	var edges []edgeRecord
	parser := NewWaySegmentParserBuilder(na, dir).
		SetWayFilter(func(way *reader.ReaderWay) bool { return true }).
		SetSplitNodeFilter(func(node *reader.ReaderNode) bool {
			return node.HasTag("barrier")
		}).
		SetEdgeHandler(func(from, to int, pl *util.PointList, way *reader.ReaderWay, nodeTags []map[string]any) {
			edges = append(edges, edgeRecord{
				from:    from,
				to:      to,
				points:  pl.Size(),
				wayID:   way.GetID(),
				barrier: way.GetTag("gh:barrier_edge") != "",
			})
		}).
		Build()

	err := parser.ReadOSM("testdata/test-osm5.xml")
	require.NoError(t, err)

	// Count barrier and non-barrier edges
	barrierCount := 0
	for _, e := range edges {
		if e.barrier {
			barrierCount++
		}
	}
	t.Logf("Total edges: %d, barrier edges: %d", len(edges), barrierCount)
}

func TestNodeClassification(t *testing.T) {
	na := newTestNodeAccess(false)
	dir := storage.NewRAMDirectory("", true)
	nd := NewOSMNodeData(na, dir)

	// Node seen at end of first way
	nd.SetOrUpdateNodeType(100, EndNode, func(prev int64) int64 { return JunctionNode })
	assert.Equal(t, EndNode, nd.GetID(100))

	// Same node seen at end of second way → connection
	nd.SetOrUpdateNodeType(100, EndNode, func(prev int64) int64 {
		if prev == EndNode {
			return ConnectionNode
		}
		return JunctionNode
	})
	assert.Equal(t, ConnectionNode, nd.GetID(100))

	// Same node seen at middle of third way → junction
	nd.SetOrUpdateNodeType(100, IntermediateNode, func(prev int64) int64 { return JunctionNode })
	assert.Equal(t, JunctionNode, nd.GetID(100))

	// Unknown node returns EmptyNode
	assert.Equal(t, EmptyNode, nd.GetID(999))
}

func TestTowerPillarConversion(t *testing.T) {
	assert.True(t, IsTowerNode(-3))
	assert.True(t, IsTowerNode(-100))
	assert.False(t, IsTowerNode(-2))
	assert.False(t, IsTowerNode(0))
	assert.False(t, IsTowerNode(3))

	assert.True(t, IsPillarNode(3))
	assert.True(t, IsPillarNode(100))
	assert.False(t, IsPillarNode(2))
	assert.False(t, IsPillarNode(0))
	assert.False(t, IsPillarNode(-3))

	na := newTestNodeAccess(false)
	dir := storage.NewRAMDirectory("", true)
	nd := NewOSMNodeData(na, dir)

	// Tower: index 0 → ID -3, index 1 → ID -4
	assert.Equal(t, int64(-3), nd.TowerNodeToID(0))
	assert.Equal(t, int64(-4), nd.TowerNodeToID(1))
	assert.Equal(t, 0, nd.IDToTowerNode(-3))
	assert.Equal(t, 1, nd.IDToTowerNode(-4))

	// Pillar: index 0 → ID 3, index 1 → ID 4
	assert.Equal(t, int64(3), nd.PillarNodeToID(0))
	assert.Equal(t, int64(4), nd.PillarNodeToID(1))
	assert.Equal(t, int64(0), nd.IDToPillarNode(3))
	assert.Equal(t, int64(1), nd.IDToPillarNode(4))
}

func TestSplitNodes(t *testing.T) {
	na := newTestNodeAccess(false)
	dir := storage.NewRAMDirectory("", true)
	nd := NewOSMNodeData(na, dir)

	assert.False(t, nd.IsSplitNode(42))
	assert.True(t, nd.SetSplitNode(42))
	assert.True(t, nd.IsSplitNode(42))
	// Setting again returns false (already set)
	assert.False(t, nd.SetSplitNode(42))

	nd.UnsetSplitNode(42)
	assert.False(t, nd.IsSplitNode(42))
}

func TestAddCoordinatesIfMapped(t *testing.T) {
	na := newTestNodeAccess(false)
	dir := storage.NewRAMDirectory("", true)
	nd := NewOSMNodeData(na, dir)

	// Unmapped node → returns EmptyNode
	nodeType := nd.AddCoordinatesIfMapped(100, 51.0, 9.0, func() float64 { return 0 })
	assert.Equal(t, EmptyNode, nodeType)

	// Map as junction → add coordinates → becomes tower
	nd.SetOrUpdateNodeType(100, JunctionNode, nil)
	nodeType = nd.AddCoordinatesIfMapped(100, 51.0, 9.0, func() float64 { return 0 })
	assert.Equal(t, JunctionNode, nodeType)
	assert.True(t, IsTowerNode(nd.GetID(100)))

	// Map as intermediate → add coordinates → becomes pillar
	nd.SetOrUpdateNodeType(200, IntermediateNode, func(prev int64) int64 { return JunctionNode })
	nodeType = nd.AddCoordinatesIfMapped(200, 52.0, 10.0, func() float64 { return 0 })
	assert.Equal(t, IntermediateNode, nodeType)
	assert.True(t, IsPillarNode(nd.GetID(200)))

	// Verify coordinates are stored
	pt := nd.GetCoordinates(nd.GetID(100))
	require.NotNil(t, pt)
	assert.InDelta(t, 51.0, pt.Lat, 0.001)
	assert.InDelta(t, 9.0, pt.Lon, 0.001)
}

func TestNodeTags(t *testing.T) {
	na := newTestNodeAccess(false)
	dir := storage.NewRAMDirectory("", true)
	nd := NewOSMNodeData(na, dir)

	// No tags initially
	assert.Nil(t, nd.GetTags(42))

	// Set tags
	tags := map[string]any{"barrier": "gate", "name": "test"}
	nd.SetTags(42, tags)
	got := nd.GetTags(42)
	assert.Equal(t, "gate", got["barrier"])
	assert.Equal(t, "test", got["name"])

	// Setting tags twice panics
	assert.Panics(t, func() {
		nd.SetTags(42, map[string]any{"foo": "bar"})
	})
}

func TestWaySegmentParserFilterRejectsWay(t *testing.T) {
	na := newTestNodeAccess(false)
	dir := storage.NewRAMDirectory("", true)

	var edges []edgeRecord
	parser := NewWaySegmentParserBuilder(na, dir).
		SetWayFilter(func(way *reader.ReaderWay) bool {
			// Reject all ways
			return false
		}).
		SetEdgeHandler(func(from, to int, pl *util.PointList, way *reader.ReaderWay, nodeTags []map[string]any) {
			edges = append(edges, edgeRecord{from: from, to: to})
		}).
		Build()

	err := parser.ReadOSM("testdata/test-osm.xml")
	require.NoError(t, err)
	assert.Empty(t, edges, "no edges when all ways are filtered out")
}
