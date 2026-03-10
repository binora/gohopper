package osm

import (
	"math"
	"testing"

	"gohopper/core/reader"
	"gohopper/core/routing"
	"gohopper/core/routing/ev"
	"gohopper/core/routing/parsers"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testGraph holds references needed by test assertions.
type testGraph struct {
	graph      *storage.BaseGraph
	em         *routingutil.EncodingManager
	carAccess  ev.BooleanEncodedValue
	carSpeed   ev.DecimalEncodedValue
	ferrySpeed ev.DecimalEncodedValue
}

func buildTestGraph(t *testing.T, osmFile string) *testGraph {
	t.Helper()

	em := routingutil.Start().
		Add(ev.VehicleAccessCreate("car")).
		Add(ev.VehicleSpeedCreate("car", 7, 2, true)).
		AddTurnCostEncodedValue(ev.TurnCostCreate("car", 1)).
		Add(ev.RoundaboutCreate()).
		Add(ev.RoadClassCreate()).
		Add(ev.RoadClassLinkCreate()).
		Add(ev.RoadEnvironmentCreate()).
		Add(ev.MaxSpeedCreate()).
		Add(ev.RoadAccessCreate()).
		Add(ev.FerrySpeedCreate()).
		Add(ev.OSMWayIDCreate()).
		Build()

	dir := storage.NewRAMDirectory("", false)
	graph := storage.NewBaseGraphBuilder(em.BytesForFlags).
		SetDir(dir).
		SetWithTurnCosts(em.NeedsTurnCostsSupport()).
		Build()
	graph.Create(100)

	osmParsers := buildCarParsers(em)
	config := routing.NewOSMReaderConfig()
	rdr := NewOSMReader(graph, osmParsers, config)
	err := rdr.ReadGraph(osmFile)
	require.NoError(t, err)

	return &testGraph{
		graph:      graph,
		em:         em,
		carAccess:  em.GetBooleanEncodedValue(ev.VehicleAccessKey("car")),
		carSpeed:   em.GetDecimalEncodedValue(ev.VehicleSpeedKey("car")),
		ferrySpeed: em.GetDecimalEncodedValue(ev.FerrySpeedKey),
	}
}

func buildCarParsers(em *routingutil.EncodingManager) *routing.OSMParsers {
	p := routing.NewOSMParsers()
	p.AddWayTagParser(parsers.NewOSMRoundaboutParser(em.GetBooleanEncodedValue(ev.RoundaboutKey)))
	p.AddWayTagParser(parsers.NewOSMRoadClassParser(em.GetEncodedValue(ev.RoadClassKey).(*ev.EnumEncodedValue[ev.RoadClass])))
	p.AddWayTagParser(parsers.NewOSMRoadClassLinkParser(em.GetBooleanEncodedValue(ev.RoadClassLinkKey)))
	p.AddWayTagParser(parsers.NewOSMRoadEnvironmentParser(em.GetEncodedValue(ev.RoadEnvironmentKey).(*ev.EnumEncodedValue[ev.RoadEnvironment])))
	p.AddWayTagParser(parsers.NewOSMMaxSpeedParser(em.GetDecimalEncodedValue(ev.MaxSpeedKey)))
	p.AddWayTagParser(parsers.NewOSMRoadAccessParser(
		em.GetEncodedValue(ev.RoadAccessKey).(*ev.EnumEncodedValue[ev.RoadAccess]),
		parsers.ToOSMRestrictions(routingutil.TransportationModeCar),
	))
	p.AddWayTagParser(parsers.NewOSMWayIDParser(em.GetIntEncodedValue(ev.OSMWayIDKey)))
	p.AddWayTagParser(parsers.NewFerrySpeedCalculator(em.GetDecimalEncodedValue(ev.FerrySpeedKey)))
	p.AddWayTagParser(parsers.NewCarAccessParser(em, true, true))
	p.AddWayTagParser(parsers.NewCarAverageSpeedParser(em))
	return p
}

// getNodeByLat returns the internal node ID whose latitude is closest to lat.
func getNodeByLat(g *storage.BaseGraph, lat float64) int {
	na := g.GetNodeAccess()
	best := -1
	bestDiff := math.MaxFloat64
	for i := 0; i < g.GetNodes(); i++ {
		diff := math.Abs(na.GetLat(i) - lat)
		if diff < bestDiff {
			bestDiff = diff
			best = i
		}
	}
	return best
}

// getNeighbors returns the set of adj nodes reachable from baseNode using the given explorer.
func getNeighbors(graph *storage.BaseGraph, node int, filter routingutil.EdgeFilter) []int {
	explorer := graph.CreateEdgeExplorer(filter)
	iter := explorer.SetBaseNode(node)
	var neighbors []int
	for iter.Next() {
		neighbors = append(neighbors, iter.GetAdjNode())
	}
	return neighbors
}

func TestOSMReaderBasic(t *testing.T) {
	tg := buildTestGraph(t, "testdata/test-osm.xml")
	g := tg.graph

	// test-osm.xml: nodes 10 (51.25), 20 (52), 30 (51.2), 50 (49)
	// node 40 (51.25) is a pillar node in way 11, not a tower node
	// node 35, 41, 45 are unused
	assert.Equal(t, 4, g.GetNodes())

	n10 := getNodeByLat(g, 51.2492152)
	n20 := getNodeByLat(g, 52)
	n30 := getNodeByLat(g, 51.2)
	n50 := getNodeByLat(g, 49)
	require.True(t, n10 >= 0)
	require.True(t, n20 >= 0)
	require.True(t, n30 >= 0)
	require.True(t, n50 >= 0)

	// All 4 nodes should be distinct
	nodes := map[int]bool{n10: true, n20: true, n30: true, n50: true}
	assert.Equal(t, 4, len(nodes), "all nodes should be distinct")

	na := g.GetNodeAccess()
	assert.InDelta(t, 51.2492152, na.GetLat(n10), 1e-5)
	assert.InDelta(t, 52.0, na.GetLat(n20), 1e-5)
	assert.InDelta(t, 51.2, na.GetLat(n30), 1e-5)
	assert.InDelta(t, 49.0, na.GetLat(n50), 1e-5)

	// Check car accessibility — all edges should be bidirectional for car
	outFilter := routingutil.OutEdges(tg.carAccess)
	allFilter := routingutil.AllAccessEdges(tg.carAccess)

	// n10 outgoing: should reach n20
	n10Neighbors := getNeighbors(g, n10, outFilter.Accept)
	assert.Contains(t, n10Neighbors, n20)

	// n30 outgoing: should reach n20
	n30Neighbors := getNeighbors(g, n30, outFilter.Accept)
	assert.Contains(t, n30Neighbors, n20)

	// n20 should have 3 edges (to n10, n30, n50) with all-access filter
	n20Neighbors := getNeighbors(g, n20, allFilter.Accept)
	assert.Equal(t, 3, len(n20Neighbors))

	// Check edge distances from n20
	explorer := g.CreateEdgeExplorer(allFilter.Accept)
	iter := explorer.SetBaseNode(n20)
	for iter.Next() {
		d := iter.GetDistance()
		assert.True(t, d > 0, "distance should be positive, got %f to node %d", d, iter.GetAdjNode())
	}
}

func TestOSMReaderOneWay(t *testing.T) {
	tg := buildTestGraph(t, "testdata/test-osm2.xml")
	g := tg.graph

	outFilter := routingutil.OutEdges(tg.carAccess)

	// Way 10: 10→20→30 motorway oneway=true
	n10 := getNodeByLat(g, 51.2492152)
	n20 := getNodeByLat(g, 52)
	n30 := getNodeByLat(g, 51.2)

	// n10 should have 1 outgoing car edge (to n20, from way 10 being oneway)
	n10Out := getNeighbors(g, n10, outFilter.Accept)
	assert.Equal(t, 1, len(n10Out), "n10 should have 1 outgoing")

	// n20 should have outgoing edges (way 10 goes 10→20→30, oneway)
	n20Out := getNeighbors(g, n20, outFilter.Accept)
	assert.True(t, len(n20Out) >= 1, "n20 should have outgoing edges")

	// n30 should have 0 outgoing car edges (all are oneway towards it)
	n30Out := getNeighbors(g, n30, outFilter.Accept)
	assert.Equal(t, 0, len(n30Out), "n30 should have 0 outgoing")

	_ = n30
}

func TestOSMReaderFerry(t *testing.T) {
	tg := buildTestGraph(t, "testdata/test-osm2.xml")
	g := tg.graph

	// Way 16: 80→90 ferry without duration → slow default speed
	n80 := getNodeByLat(g, 54.1)
	allFilter := routingutil.AllAccessEdges(tg.carAccess)
	explorer := g.CreateEdgeExplorer(allFilter.Accept)
	iter := explorer.SetBaseNode(n80)
	found := false
	for iter.Next() {
		speed := tg.ferrySpeed.GetDecimal(false, iter.GetEdge(), g.Store)
		if speed > 0 {
			// Ferry without duration defaults to small speed
			assert.True(t, speed <= 10, "ferry without duration should have slow speed, got %f", speed)
			found = true
		}
	}
	assert.True(t, found, "should find ferry edge from n80")
}

func TestOSMReaderMissingNode(t *testing.T) {
	tg := buildTestGraph(t, "testdata/test-osm4.xml")
	g := tg.graph

	// test-osm4.xml: nodes 10 (51.2492152) and 30 (51.2)
	// Way 10: 10→[missing 20]→30 motorway
	// Missing node 20 should be handled gracefully
	assert.Equal(t, 2, g.GetNodes())
	assert.Equal(t, 1, g.GetEdges())

	n10 := getNodeByLat(g, 51.2492152)
	n30 := getNodeByLat(g, 51.2)
	assert.NotEqual(t, n10, n30)

	// They should still be connected
	allFilter := routingutil.AllAccessEdges(tg.carAccess)
	neighbors := getNeighbors(g, n10, allFilter.Accept)
	assert.Contains(t, neighbors, n30)
}

func TestOSMReaderBarriers(t *testing.T) {
	tg := buildTestGraph(t, "testdata/test-barriers.xml")
	g := tg.graph

	// test-barriers.xml:
	//      b      b
	// 10-20-30- 50 -60
	//  |       |  |
	//  40- - -/  70
	//             |
	//            80
	// Barrier at node 20 (pillar) creates split → extra node
	// Barrier at node 50 (tower) is ignored
	assert.Equal(t, 7, g.GetNodes())
	assert.Equal(t, 7, g.GetEdges())

	n10 := getNodeByLat(g, 51)
	n60 := getNodeByLat(g, 56)

	// n10 should have 2 outgoing edges (to n20-split and via n40 to n30)
	allFilter := routingutil.AllAccessEdges(tg.carAccess)
	n10Neighbors := getNeighbors(g, n10, allFilter.Accept)
	assert.Equal(t, 2, len(n10Neighbors), "n10 should have 2 neighbors")

	// n60 should have 1 edge (to n50)
	n60Neighbors := getNeighbors(g, n60, allFilter.Accept)
	assert.Equal(t, 1, len(n60Neighbors), "n60 should have 1 neighbor")
}

func TestOSMReaderNegativeIds(t *testing.T) {
	em := routingutil.Start().
		Add(ev.VehicleAccessCreate("car")).
		Add(ev.VehicleSpeedCreate("car", 7, 2, true)).
		Add(ev.RoundaboutCreate()).
		Add(ev.RoadClassCreate()).
		Add(ev.RoadClassLinkCreate()).
		Add(ev.RoadEnvironmentCreate()).
		Add(ev.MaxSpeedCreate()).
		Add(ev.RoadAccessCreate()).
		Add(ev.FerrySpeedCreate()).
		Add(ev.OSMWayIDCreate()).
		Build()

	dir := storage.NewRAMDirectory("", false)
	graph := storage.NewBaseGraphBuilder(em.BytesForFlags).
		SetDir(dir).
		Build()
	graph.Create(100)

	osmParsers := buildCarParsers(em)
	config := routing.NewOSMReaderConfig()
	rdr := NewOSMReader(graph, osmParsers, config)

	// Negative IDs should cause an error or panic
	assert.Panics(t, func() {
		rdr.ReadGraph("testdata/test-osm-negative-ids.xml")
	})
}

// helper to assert a reader.ReaderWay is accepted by the parsers
func TestAcceptWay(t *testing.T) {
	em := routingutil.Start().
		Add(ev.VehicleAccessCreate("car")).
		Add(ev.VehicleSpeedCreate("car", 7, 2, true)).
		Add(ev.RoundaboutCreate()).
		Add(ev.RoadClassCreate()).
		Add(ev.RoadClassLinkCreate()).
		Add(ev.RoadEnvironmentCreate()).
		Add(ev.MaxSpeedCreate()).
		Add(ev.RoadAccessCreate()).
		Add(ev.FerrySpeedCreate()).
		Add(ev.OSMWayIDCreate()).
		Build()

	osmParsers := buildCarParsers(em)
	config := routing.NewOSMReaderConfig()
	dir := storage.NewRAMDirectory("", false)
	graph := storage.NewBaseGraphBuilder(em.BytesForFlags).SetDir(dir).Build()
	graph.Create(10)
	rdr := NewOSMReader(graph, osmParsers, config)

	// Way with highway=motorway and nodes → accepted
	way := reader.NewReaderWay(1)
	way.Nodes = []int64{1, 2}
	way.SetTag("highway", "motorway")
	assert.True(t, rdr.acceptWay(way))

	// Way without tags → rejected
	wayNoTags := reader.NewReaderWay(2)
	wayNoTags.Nodes = []int64{1, 2}
	assert.False(t, rdr.acceptWay(wayNoTags))

	// Way with less than 2 nodes → rejected
	wayShort := reader.NewReaderWay(3)
	wayShort.Nodes = []int64{1}
	wayShort.SetTag("highway", "motorway")
	assert.False(t, rdr.acceptWay(wayShort))
}

func TestAcceptWayWithIgnoredHighways(t *testing.T) {
	em := routingutil.Start().
		Add(ev.VehicleAccessCreate("car")).
		Add(ev.VehicleSpeedCreate("car", 7, 2, true)).
		Add(ev.RoundaboutCreate()).
		Add(ev.RoadClassCreate()).
		Add(ev.RoadClassLinkCreate()).
		Add(ev.RoadEnvironmentCreate()).
		Add(ev.MaxSpeedCreate()).
		Add(ev.RoadAccessCreate()).
		Add(ev.FerrySpeedCreate()).
		Add(ev.OSMWayIDCreate()).
		Build()

	osmParsers := buildCarParsers(em)
	osmParsers.AddIgnoredHighway("footway")
	osmParsers.AddIgnoredHighway("cycleway")
	osmParsers.AddIgnoredHighway("path")

	config := routing.NewOSMReaderConfig()
	dir := storage.NewRAMDirectory("", false)
	graph := storage.NewBaseGraphBuilder(em.BytesForFlags).SetDir(dir).Build()
	graph.Create(10)
	rdr := NewOSMReader(graph, osmParsers, config)

	// highway=residential → accepted (not in ignored list)
	way := reader.NewReaderWay(1)
	way.Nodes = []int64{1, 2}
	way.SetTag("highway", "residential")
	assert.True(t, rdr.acceptWay(way))

	// highway=footway → rejected (in ignored list)
	wayFoot := reader.NewReaderWay(2)
	wayFoot.Nodes = []int64{1, 2}
	wayFoot.SetTag("highway", "footway")
	assert.False(t, rdr.acceptWay(wayFoot))

	// highway=cycleway → rejected (in ignored list)
	wayCycle := reader.NewReaderWay(3)
	wayCycle.Nodes = []int64{1, 2}
	wayCycle.SetTag("highway", "cycleway")
	assert.False(t, rdr.acceptWay(wayCycle))

	// highway=path → rejected (in ignored list)
	wayPath := reader.NewReaderWay(4)
	wayPath.Nodes = []int64{1, 2}
	wayPath.SetTag("highway", "path")
	assert.False(t, rdr.acceptWay(wayPath))

	// highway=motorway → accepted (not in ignored list)
	wayMotor := reader.NewReaderWay(5)
	wayMotor.Nodes = []int64{1, 2}
	wayMotor.SetTag("highway", "motorway")
	assert.True(t, rdr.acceptWay(wayMotor))
}
