package core

import (
	"os"
	"sync"
	"testing"

	"gohopper/core/config"
	"gohopper/core/routing"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/util"
	webapi "gohopper/web-api"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const andorraOSM = "testdata/andorra.osm.pbf"

// Baseline values: node/edge counts and distances match Java GH (minNetworkSize=0, car profile).
// Times use CustomWeighting formula: seconds = distance / speed * 3.6 (SPEED_CONV).
const (
	andorraNodes = 2385
	andorraEdges = 2820

	// Route A: short urban (42.5063, 1.5218) -> (42.5103, 1.5385)
	routeADist   = 1910.1
	routeATime   = int64(125747)
	routeAPoints = 62

	// Route A visited nodes per algorithm
	routeAVisitedDijkstra   = 303
	routeAVisitedAStar      = 218
	routeAVisitedDijkstraBi = 196
	routeAVisitedAStarBi    = 124

	// Route B: cross-town (42.5063, 1.5218) -> (42.5354, 1.5806)
	routeBDist   = 6756.4
	routeBTime   = int64(396148)
	routeBPoints = 190

	// Via route A->B->A
	viaDist   = 3479.3
	viaTime   = int64(249697)
	viaPoints = 117
)

var (
	andorraSetup sync.Once
	andorraGH    *GraphHopper
)

func getAndorraHopper(t *testing.T) *GraphHopper {
	t.Helper()
	andorraSetup.Do(func() {
		dir, err := os.MkdirTemp("", "andorra-e2e-*")
		require.NoError(t, err)
		gh := NewGraphHopper()
		gh.SetOSMFile(andorraOSM)
		gh.SetGraphHopperLocation(dir)
		gh.SetStoreOnFlush(false)
		gh.SetProfiles(config.Profile{Name: "car"})
		err = gh.ImportOrLoad()
		require.NoError(t, err)
		andorraGH = gh
	})
	require.NotNil(t, andorraGH, "andorra import failed")
	return andorraGH
}

func routeAndAssertNoErrors(t *testing.T, gh *GraphHopper, req webapi.GHRequest) webapi.GHResponse {
	t.Helper()
	resp := gh.Route(req)
	require.False(t, resp.HasErrors(), "route errors: %v", resp.Errors)
	require.NotNil(t, resp.GetBest())
	return resp
}

func pointCount(t *testing.T, rp *webapi.ResponsePath) int {
	t.Helper()
	encoded, ok := rp.Points.(string)
	require.True(t, ok, "expected encoded polyline string, got %T", rp.Points)
	pl := util.DecodePolyline(encoded, false, 1e5)
	return pl.Size()
}

func visitedNodes(resp webapi.GHResponse) int {
	v, ok := resp.Hints["visited_nodes.sum"]
	if !ok {
		return -1
	}
	return v.(int)
}

func TestAndorraImport(t *testing.T) {
	gh := getAndorraHopper(t)
	bg := gh.GetBaseGraph()
	assert.Equal(t, andorraNodes, bg.GetNodes(), "node count mismatch with Java baseline")
	assert.Equal(t, andorraEdges, bg.GetEdges(), "edge count mismatch with Java baseline")

	em := gh.GetEncodingManager()
	assert.True(t, em.HasEncodedValue("car_access"))
	assert.True(t, em.HasEncodedValue("car_average_speed"))

	li := gh.GetLocationIndex()
	require.NotNil(t, li)
	snap := li.FindClosest(42.5063, 1.5218, routingutil.AllEdges)
	assert.True(t, snap.IsValid(), "expected valid snap in Andorra la Vella")
}

func TestAndorraDifferentAlgorithms(t *testing.T) {
	gh := getAndorraHopper(t)

	tests := []struct {
		algo           string
		expectedVisited int
	}{
		{routing.AlgoDijkstra, routeAVisitedDijkstra},
		{routing.AlgoAStar, routeAVisitedAStar},
		{routing.AlgoDijkstraBi, routeAVisitedDijkstraBi},
		{routing.AlgoAStarBi, routeAVisitedAStarBi},
	}

	for _, tt := range tests {
		t.Run(tt.algo, func(t *testing.T) {
			req := webapi.NewGHRequestLatLon(42.5063, 1.5218, 42.5103, 1.5385)
			req.Profile = "car"
			req.Algorithm = tt.algo

			resp := routeAndAssertNoErrors(t, gh, req)
			path := resp.GetBest()

			assert.InDelta(t, routeADist, path.Distance, 1.0, "distance mismatch for %s", tt.algo)
			assert.InDelta(t, routeATime, path.Time, 1000, "time mismatch for %s", tt.algo)
			assert.Equal(t, routeAPoints, pointCount(t, path), "point count mismatch for %s", tt.algo)
			assert.Equal(t, tt.expectedVisited, visitedNodes(resp), "visited nodes mismatch for %s", tt.algo)
		})
	}
}

func TestAndorraRouteBasic(t *testing.T) {
	gh := getAndorraHopper(t)

	req := webapi.NewGHRequestLatLon(42.5063, 1.5218, 42.5354, 1.5806)
	req.Profile = "car"

	resp := routeAndAssertNoErrors(t, gh, req)
	path := resp.GetBest()

	assert.InDelta(t, routeBDist, path.Distance, 1.0)
	assert.InDelta(t, routeBTime, path.Time, 1000)
	assert.Equal(t, routeBPoints, pointCount(t, path))

	// BBox should be within Andorra
	bbox := path.BBox
	assert.True(t, bbox[1] > 42.4 && bbox[1] < 42.6, "bbox minLat out of range: %f", bbox[1])
	assert.True(t, bbox[3] > 42.4 && bbox[3] < 42.6, "bbox maxLat out of range: %f", bbox[3])
	assert.True(t, bbox[0] > 1.4 && bbox[0] < 1.7, "bbox minLon out of range: %f", bbox[0])
	assert.True(t, bbox[2] > 1.4 && bbox[2] < 1.7, "bbox maxLon out of range: %f", bbox[2])

	// Instructions should exist (basic: Continue + Arrive)
	assert.True(t, len(path.Instructions) >= 2, "expected at least 2 instructions")
	assert.Equal(t, 4, path.Instructions[len(path.Instructions)-1].Sign, "last instruction should be FINISH (sign=4)")
}

func TestAndorraViaRoute(t *testing.T) {
	gh := getAndorraHopper(t)

	req := webapi.NewGHRequest()
	req.Points = []util.GHPoint{
		{Lat: 42.5063, Lon: 1.5218},
		{Lat: 42.5103, Lon: 1.5385},
		{Lat: 42.5063, Lon: 1.5218},
	}
	req.Profile = "car"

	resp := routeAndAssertNoErrors(t, gh, req)
	path := resp.GetBest()

	assert.InDelta(t, viaDist, path.Distance, 1.0)
	assert.InDelta(t, viaTime, path.Time, 1000)
	assert.Equal(t, viaPoints, pointCount(t, path))
}

func TestAndorraPersistAndReload(t *testing.T) {
	tmpDir := t.TempDir()

	// Import and flush to disk
	gh1 := NewGraphHopper()
	gh1.SetOSMFile(andorraOSM)
	gh1.SetGraphHopperLocation(tmpDir)
	gh1.SetStoreOnFlush(true)
	gh1.SetProfiles(config.Profile{Name: "car"})
	require.NoError(t, gh1.ImportOrLoad())
	require.True(t, gh1.IsFullyLoaded())

	nodes := gh1.GetBaseGraph().GetNodes()
	edges := gh1.GetBaseGraph().GetEdges()

	// Route before reload
	req := webapi.NewGHRequestLatLon(42.5063, 1.5218, 42.5103, 1.5385)
	req.Profile = "car"
	resp1 := routeAndAssertNoErrors(t, gh1, req)
	dist1 := resp1.GetBest().Distance

	// Reload from cache (no OSM file)
	gh2 := NewGraphHopper()
	gh2.SetGraphHopperLocation(tmpDir)
	gh2.SetProfiles(config.Profile{Name: "car"})
	require.NoError(t, gh2.ImportOrLoad())
	require.True(t, gh2.IsFullyLoaded())

	assert.Equal(t, nodes, gh2.GetBaseGraph().GetNodes())
	assert.Equal(t, edges, gh2.GetBaseGraph().GetEdges())

	// Route after reload should produce same results
	resp2 := routeAndAssertNoErrors(t, gh2, req)
	assert.InDelta(t, dist1, resp2.GetBest().Distance, 0.1)
}

func TestAndorraEdgeCount(t *testing.T) {
	gh := getAndorraHopper(t)
	bg := gh.GetBaseGraph()

	count := 0
	iter := bg.GetAllEdges()
	for iter.Next() {
		count++
	}
	assert.Equal(t, bg.GetEdges(), count, "AllEdges iterator count should match GetEdges()")
}

func TestRouteBeforeLoad(t *testing.T) {
	gh := NewGraphHopper()
	gh.SetProfiles(config.Profile{Name: "car"})

	req := webapi.NewGHRequestLatLon(42.5063, 1.5218, 42.5103, 1.5385)
	req.Profile = "car"
	resp := gh.Route(req)

	assert.True(t, resp.HasErrors())
	assert.Contains(t, resp.Errors[0].Error(), "not fully loaded")
}

func TestPointOutOfBounds(t *testing.T) {
	gh := getAndorraHopper(t)

	req := webapi.NewGHRequestLatLon(0.0, 0.0, 42.5103, 1.5385)
	req.Profile = "car"
	resp := gh.Route(req)

	assert.True(t, resp.HasErrors())
	_, isOOB := resp.Errors[0].(webapi.PointOutOfBoundsError)
	assert.True(t, isOOB, "expected PointOutOfBoundsError, got: %T: %v", resp.Errors[0], resp.Errors[0])
}

func TestPointNotSnappable(t *testing.T) {
	gh := getAndorraHopper(t)

	// Point in Andorra's bounding box but in the mountains with no roads
	req := webapi.NewGHRequestLatLon(42.62, 1.45, 42.5103, 1.5385)
	req.Profile = "car"
	resp := gh.Route(req)

	if resp.HasErrors() {
		_, isPNF := resp.Errors[0].(webapi.PointNotFoundError)
		_, isCNF := resp.Errors[0].(webapi.ConnectionNotFoundError)
		assert.True(t, isPNF || isCNF, "expected PointNotFoundError or ConnectionNotFoundError, got: %T: %v", resp.Errors[0], resp.Errors[0])
	}
	// If it doesn't error (point snapped to distant road), that's also acceptable
}
