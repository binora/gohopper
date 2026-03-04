package core

import (
	"testing"

	routingutil "gohopper/core/routing/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportAndRoute(t *testing.T) {
	tmpDir := t.TempDir()

	gh := NewGraphHopper()
	gh.SetOSMFile("reader/osm/testdata/test-osm.xml")
	gh.SetGraphHopperLocation(tmpDir)

	err := gh.ImportOrLoad()
	require.NoError(t, err)
	require.True(t, gh.IsFullyLoaded())

	bg := gh.GetBaseGraph()
	require.NotNil(t, bg)
	assert.True(t, bg.GetNodes() > 0, "expected nodes > 0, got %d", bg.GetNodes())
	assert.True(t, bg.GetEdges() > 0, "expected edges > 0, got %d", bg.GetEdges())

	// test-osm.xml: 4 tower nodes, 3 edges (way 10: 10→20→30 and way 11: 20→40→50)
	assert.Equal(t, 4, bg.GetNodes())
	assert.Equal(t, 3, bg.GetEdges())

	// Verify LocationIndex works
	locIndex := gh.GetLocationIndex()
	require.NotNil(t, locIndex)

	// Snap to node 20 at (52, 9)
	snap := locIndex.FindClosest(52.0, 9.0, routingutil.AllEdges)
	assert.True(t, snap.IsValid(), "expected valid snap near (52, 9)")

	// Snap to node 50 at (49, 10)
	snap2 := locIndex.FindClosest(49.0, 10.0, routingutil.AllEdges)
	assert.True(t, snap2.IsValid(), "expected valid snap near (49, 10)")

	// Verify EncodingManager is set
	em := gh.GetEncodingManager()
	require.NotNil(t, em)
	assert.True(t, em.HasEncodedValue("car_access"))
	assert.True(t, em.HasEncodedValue("car_average_speed"))
}

func TestImportPersistsAndReloads(t *testing.T) {
	tmpDir := t.TempDir()

	// First: import
	gh1 := NewGraphHopper()
	gh1.SetOSMFile("reader/osm/testdata/test-osm.xml")
	gh1.SetGraphHopperLocation(tmpDir)

	err := gh1.ImportOrLoad()
	require.NoError(t, err)
	require.True(t, gh1.IsFullyLoaded())
	nodes := gh1.GetBaseGraph().GetNodes()
	edges := gh1.GetBaseGraph().GetEdges()

	// Second: reload from disk
	gh2 := NewGraphHopper()
	gh2.SetGraphHopperLocation(tmpDir)

	err = gh2.ImportOrLoad()
	require.NoError(t, err)
	require.True(t, gh2.IsFullyLoaded())
	assert.Equal(t, nodes, gh2.GetBaseGraph().GetNodes())
	assert.Equal(t, edges, gh2.GetBaseGraph().GetEdges())
}

func TestImportOneway(t *testing.T) {
	tmpDir := t.TempDir()

	gh := NewGraphHopper()
	gh.SetOSMFile("reader/osm/testdata/test-osm2.xml")
	gh.SetGraphHopperLocation(tmpDir)
	gh.SetStoreOnFlush(false) // in-memory only

	err := gh.ImportOrLoad()
	require.NoError(t, err)

	bg := gh.GetBaseGraph()
	em := gh.GetEncodingManager()
	carAccess := em.GetBooleanEncodedValue("car_access")

	// Way 10: 10→20→30 motorway oneway=true
	// node 30 (lat 51.2) should have no outgoing car edges
	n30 := -1
	na := bg.GetNodeAccess()
	for i := 0; i < bg.GetNodes(); i++ {
		lat := na.GetLat(i)
		if lat > 51.19 && lat < 51.21 {
			n30 = i
			break
		}
	}
	require.True(t, n30 >= 0, "should find node at lat ~51.2")

	outFilter := routingutil.OutEdges(carAccess)
	explorer := bg.CreateEdgeExplorer(outFilter.Accept)
	iter := explorer.SetBaseNode(n30)
	outCount := 0
	for iter.Next() {
		outCount++
	}
	assert.Equal(t, 0, outCount, "node 30 should have 0 outgoing car edges due to oneway")
}

func TestImportBarriers(t *testing.T) {
	tmpDir := t.TempDir()

	gh := NewGraphHopper()
	gh.SetOSMFile("reader/osm/testdata/test-barriers.xml")
	gh.SetGraphHopperLocation(tmpDir)
	gh.SetStoreOnFlush(false)

	err := gh.ImportOrLoad()
	require.NoError(t, err)

	bg := gh.GetBaseGraph()
	assert.Equal(t, 7, bg.GetNodes())
	assert.Equal(t, 7, bg.GetEdges())
}
