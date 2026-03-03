package parsers

import (
	"testing"

	"gohopper/core/reader"
	"gohopper/core/routing/ev"
	emutil "gohopper/core/routing/util"

	"github.com/stretchr/testify/assert"
)

func createCarEM() *emutil.EncodingManager {
	return emutil.Start().
		Add(ev.VehicleAccessCreate("car")).
		Add(ev.VehicleSpeedCreate("car", 7, 2, true)).
		AddTurnCostEncodedValue(ev.TurnCostCreate("car", 1)).
		Add(ev.RoundaboutCreate()).
		Add(ev.FerrySpeedCreate()).
		Build()
}

func TestAccess(t *testing.T) {
	em := createCarEM()
	parser := NewCarAccessParser(em, true, true)

	way := reader.NewReaderWay(1)
	assert.True(t, parser.GetAccess(way).CanSkip())
	way.SetTag("highway", "service")
	assert.True(t, parser.GetAccess(way).IsWay())
	way.SetTag("access", "no")
	assert.True(t, parser.GetAccess(way).CanSkip())

	way.ClearTags()
	way.SetTag("highway", "track")
	assert.True(t, parser.GetAccess(way).IsWay())

	way.SetTag("motorcar", "no")
	assert.True(t, parser.GetAccess(way).CanSkip())

	// Grade 2 allowed.
	way.ClearTags()
	way.SetTag("highway", "track")
	way.SetTag("tracktype", "grade2")
	assert.True(t, parser.GetAccess(way).IsWay())
	way.SetTag("tracktype", "grade4")
	assert.True(t, parser.GetAccess(way).CanSkip())

	way.ClearTags()
	way.SetTag("highway", "service")
	way.SetTag("access", "delivery")
	assert.True(t, parser.GetAccess(way).CanSkip())

	way.ClearTags()
	way.SetTag("highway", "unclassified")
	way.SetTag("ford", "yes")
	assert.True(t, parser.GetAccess(way).CanSkip())
	way.SetTag("motorcar", "yes")
	assert.True(t, parser.GetAccess(way).IsWay())

	way.ClearTags()
	way.SetTag("access", "yes")
	way.SetTag("motor_vehicle", "no")
	assert.True(t, parser.GetAccess(way).CanSkip())

	way.ClearTags()
	way.SetTag("highway", "service")
	way.SetTag("access", "yes")
	way.SetTag("motor_vehicle", "no")
	assert.True(t, parser.GetAccess(way).CanSkip())

	way.ClearTags()
	way.SetTag("highway", "track")
	way.SetTag("motor_vehicle", "agricultural")
	assert.True(t, parser.GetAccess(way).CanSkip())
	way.SetTag("motor_vehicle", "agricultural;forestry")
	assert.True(t, parser.GetAccess(way).CanSkip())
	way.SetTag("motor_vehicle", "forestry;agricultural")
	assert.True(t, parser.GetAccess(way).CanSkip())
	way.SetTag("motor_vehicle", "forestry;agricultural;unknown")
	assert.True(t, parser.GetAccess(way).CanSkip())
	way.SetTag("motor_vehicle", "yes;forestry;agricultural")
	assert.True(t, parser.GetAccess(way).IsWay())

	way.ClearTags()
	way.SetTag("highway", "service")
	way.SetTag("access", "no")
	way.SetTag("motorcar", "yes")
	assert.True(t, parser.GetAccess(way).IsWay())

	way.ClearTags()
	way.SetTag("highway", "service")
	way.SetTag("access", "emergency")
	assert.True(t, parser.GetAccess(way).CanSkip())

	way.ClearTags()
	way.SetTag("highway", "service")
	way.SetTag("motor_vehicle", "emergency")
	assert.True(t, parser.GetAccess(way).CanSkip())

	way.ClearTags()
	way.SetTag("highway", "service")
	way.SetTag("service", "emergency_access")
	assert.True(t, parser.GetAccess(way).CanSkip())
}

func TestMilitaryAccess(t *testing.T) {
	em := createCarEM()
	parser := NewCarAccessParser(em, true, true)

	way := reader.NewReaderWay(1)
	way.SetTag("highway", "track")
	way.SetTag("access", "military")
	assert.True(t, parser.GetAccess(way).CanSkip())
}

func TestFordAccess(t *testing.T) {
	em := createCarEM()
	parser := NewCarAccessParser(em, true, true)

	node := reader.NewReaderNode(0, 0.0, 0.0)
	node.SetTag("ford", "yes")

	way := reader.NewReaderWay(1)
	way.SetTag("highway", "unclassified")
	way.SetTag("ford", "yes")

	assert.True(t, parser.IsBlockFords())
	assert.True(t, parser.GetAccess(way).CanSkip())
	assert.True(t, parser.IsBarrier(node))

	noFordParser := NewCarAccessParser(em, false, true)
	assert.True(t, noFordParser.GetAccess(way).IsWay())
	assert.False(t, noFordParser.IsBarrier(node))
}

func TestOneway(t *testing.T) {
	em := createCarEM()
	parser := NewCarAccessParser(em, true, true)
	accessEnc := parser.GetAccessEnc()

	way := reader.NewReaderWay(1)
	way.SetTag("highway", "primary")
	edgeIntAccess := ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	edgeID := 0
	parser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.True(t, accessEnc.GetBool(false, edgeID, edgeIntAccess))
	assert.True(t, accessEnc.GetBool(true, edgeID, edgeIntAccess))

	way.SetTag("oneway", "yes")
	edgeIntAccess = ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	parser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.True(t, accessEnc.GetBool(false, edgeID, edgeIntAccess))
	assert.False(t, accessEnc.GetBool(true, edgeID, edgeIntAccess))
	way.ClearTags()

	way.SetTag("highway", "tertiary")
	edgeIntAccess = ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	parser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.True(t, accessEnc.GetBool(false, edgeID, edgeIntAccess))
	assert.True(t, accessEnc.GetBool(true, edgeID, edgeIntAccess))
	way.ClearTags()

	way.SetTag("highway", "tertiary")
	way.SetTag("vehicle:forward", "no")
	edgeIntAccess = ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	parser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.False(t, accessEnc.GetBool(false, edgeID, edgeIntAccess))
	assert.True(t, accessEnc.GetBool(true, edgeID, edgeIntAccess))
	way.ClearTags()

	way.SetTag("highway", "tertiary")
	way.SetTag("vehicle:backward", "no")
	edgeIntAccess = ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	parser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.True(t, accessEnc.GetBool(false, edgeID, edgeIntAccess))
	assert.False(t, accessEnc.GetBool(true, edgeID, edgeIntAccess))
	way.ClearTags()

	// vehicle:backward=designated is not a one-way.
	way.SetTag("highway", "tertiary")
	way.SetTag("vehicle:backward", "designated")
	edgeIntAccess = ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	parser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.True(t, accessEnc.GetBool(false, edgeID, edgeIntAccess))
	assert.True(t, accessEnc.GetBool(true, edgeID, edgeIntAccess))
}

func TestShouldBlockPrivate(t *testing.T) {
	em := createCarEM()
	parser := NewCarAccessParser(em, true, true)
	accessEnc := parser.GetAccessEnc()

	way := reader.NewReaderWay(1)
	way.SetTag("highway", "primary")
	way.SetTag("access", "private")
	edgeIntAccess := ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	edgeID := 0
	parser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.False(t, accessEnc.GetBool(false, edgeID, edgeIntAccess))

	noBlockParser := NewCarAccessParser(em, true, false)
	edgeIntAccess = ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	noBlockParser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.True(t, noBlockParser.GetAccessEnc().GetBool(false, edgeID, edgeIntAccess))
}

func TestMaxSpeed(t *testing.T) {
	em := createCarEM()
	speedParser := NewCarAverageSpeedParser(em)
	avSpeedEnc := speedParser.GetAverageSpeedEnc()

	way := reader.NewReaderWay(1)
	way.SetTag("highway", "trunk")
	way.SetTag("maxspeed", "500")
	edgeIntAccess := ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	edgeID := 0
	speedParser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.InDelta(t, 136, avSpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 1)

	way = reader.NewReaderWay(1)
	way.SetTag("highway", "primary")
	way.SetTag("maxspeed:backward", "10")
	way.SetTag("maxspeed:forward", "20")
	edgeIntAccess = ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	speedParser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.InDelta(t, 18, avSpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 1)
	assert.InDelta(t, 10, avSpeedEnc.GetDecimal(true, edgeID, edgeIntAccess), 1)

	way = reader.NewReaderWay(1)
	way.SetTag("highway", "primary")
	way.SetTag("maxspeed:forward", "20")
	edgeIntAccess = ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	speedParser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.InDelta(t, 18, avSpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 1)

	way = reader.NewReaderWay(1)
	way.SetTag("highway", "primary")
	way.SetTag("maxspeed:backward", "20")
	edgeIntAccess = ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	speedParser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.InDelta(t, 66, avSpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 1)
	assert.InDelta(t, 18, avSpeedEnc.GetDecimal(true, edgeID, edgeIntAccess), 1)

	way = reader.NewReaderWay(1)
	way.SetTag("highway", "motorway")
	way.SetTag("maxspeed", "none")
	edgeIntAccess = ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	speedParser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.InDelta(t, 136, avSpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 1)

	way = reader.NewReaderWay(1)
	way.SetTag("highway", "motorway_link")
	way.SetTag("maxspeed", "70 mph")
	edgeIntAccess = ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	speedParser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.InDelta(t, 102, avSpeedEnc.GetDecimal(true, edgeID, edgeIntAccess), 1)
}

func TestSpeed(t *testing.T) {
	em := createCarEM()
	speedParser := NewCarAverageSpeedParser(em)
	avSpeedEnc := speedParser.GetAverageSpeedEnc()
	edgeID := 0

	// Trunk with limit bigger than default.
	way := reader.NewReaderWay(1)
	way.SetTag("highway", "trunk")
	way.SetTag("maxspeed", "110")
	edgeIntAccess := ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	speedParser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.InDelta(t, 100, avSpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 1)

	way.ClearTags()
	way.SetTag("highway", "residential")
	way.SetTag("surface", "cobblestone")
	edgeIntAccess = ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	speedParser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.InDelta(t, 30, avSpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 1)

	way.ClearTags()
	way.SetTag("highway", "track")
	edgeIntAccess = ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	speedParser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.InDelta(t, 16, avSpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 1)

	way.ClearTags()
	way.SetTag("highway", "track")
	way.SetTag("tracktype", "grade1")
	edgeIntAccess = ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	speedParser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.InDelta(t, 20, avSpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 1)

	way.ClearTags()
	way.SetTag("highway", "secondary")
	way.SetTag("surface", "compacted")
	edgeIntAccess = ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	speedParser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.InDelta(t, 30, avSpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 1)

	way.ClearTags()
	way.SetTag("highway", "secondary")
	way.SetTag("motorroad", "yes")
	edgeIntAccess = ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	speedParser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.InDelta(t, 60, avSpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 1)

	way.ClearTags()
	way.SetTag("highway", "motorway")
	way.SetTag("motorroad", "yes")
	edgeIntAccess = ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	speedParser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.InDelta(t, 100, avSpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 1)

	way.ClearTags()
	way.SetTag("highway", "motorway_link")
	way.SetTag("motorroad", "yes")
	edgeIntAccess = ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	speedParser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.InDelta(t, 70, avSpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 1)
}

func TestSetSpeed(t *testing.T) {
	em := createCarEM()
	avSpeedEnc := em.GetDecimalEncodedValue(ev.VehicleSpeedKey("car"))

	edgeIntAccess := ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	edgeID := 0
	avSpeedEnc.SetDecimal(false, edgeID, edgeIntAccess, 10)
	assert.InDelta(t, 10, avSpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 0.1)
}

func TestApplyBadSurfaceSpeed(t *testing.T) {
	em := createCarEM()
	speedParser := NewCarAverageSpeedParser(em)

	way := reader.NewReaderWay(1)
	way.SetTag("highway", "secondary")
	way.SetTag("surface", "unpaved")
	assert.InDelta(t, 30, speedParser.ApplyBadSurfaceSpeed(way, 90), 0.1)
}

func TestBarrierAccess(t *testing.T) {
	em := createCarEM()
	parser := NewCarAccessParser(em, true, true)

	node := reader.NewReaderNode(1, -1, -1)
	node.SetTag("barrier", "lift_gate")
	node.SetTag("access", "yes")
	assert.False(t, parser.IsBarrier(node))

	node = reader.NewReaderNode(1, -1, -1)
	node.SetTag("barrier", "lift_gate")
	node.SetTag("bicycle", "yes")
	assert.False(t, parser.IsBarrier(node))

	node = reader.NewReaderNode(1, -1, -1)
	node.SetTag("barrier", "lift_gate")
	node.SetTag("access", "no")
	node.SetTag("motorcar", "yes")
	assert.False(t, parser.IsBarrier(node))

	node = reader.NewReaderNode(1, -1, -1)
	node.SetTag("barrier", "bollard")
	assert.True(t, parser.IsBarrier(node))

	node = reader.NewReaderNode(1, -1, -1)
	node.SetTag("barrier", "cattle_grid")
	assert.False(t, parser.IsBarrier(node))
}

func TestChainBarrier(t *testing.T) {
	em := createCarEM()
	parser := NewCarAccessParser(em, true, true)

	node := reader.NewReaderNode(1, -1, -1)
	node.SetTag("barrier", "chain")
	assert.False(t, parser.IsBarrier(node))
	node.SetTag("motor_vehicle", "no")
	assert.True(t, parser.IsBarrier(node))
	node.SetTag("motor_vehicle", "yes")
	assert.False(t, parser.IsBarrier(node))
}

func TestFerry(t *testing.T) {
	em := createCarEM()
	parser := NewCarAccessParser(em, true, true)

	way := reader.NewReaderWay(1)
	way.SetTag("route", "shuttle_train")
	way.SetTag("motorcar", "yes")
	way.SetTag("bicycle", "no")
	way.SetTag("way_distance", 50000.0)
	way.SetTag("speed_from_duration", 50/(35.0/60))
	assert.True(t, parser.GetAccess(way).IsFerry())

	way = reader.NewReaderWay(1)
	way.SetTag("route", "ferry")
	way.SetTag("motorcar", "yes")
	way.SetTag("way_distance", 100.0)
	way.SetTag("speed_from_duration", 0.1/(12.0/60))
	assert.True(t, parser.GetAccess(way).IsFerry())

	way = reader.NewReaderWay(1)
	way.SetTag("route", "ferry")
	way.SetTag("motorcar", "yes")
	way.SetTag("edge_distance", 100.0)
	assert.True(t, parser.GetAccess(way).IsFerry())

	way.ClearTags()
	way.SetTag("route", "ferry")
	assert.True(t, parser.GetAccess(way).IsFerry())
	way.SetTag("motorcar", "no")
	assert.True(t, parser.GetAccess(way).CanSkip())

	way.ClearTags()
	way.SetTag("route", "ferry")
	way.SetTag("foot", "yes")
	assert.True(t, parser.GetAccess(way).CanSkip())

	way.ClearTags()
	way.SetTag("route", "ferry")
	way.SetTag("access", "no")
	assert.True(t, parser.GetAccess(way).CanSkip())
	way.SetTag("vehicle", "yes")
	assert.True(t, parser.GetAccess(way).IsFerry())
}

func TestRailway(t *testing.T) {
	em := createCarEM()
	parser := NewCarAccessParser(em, true, true)

	way := reader.NewReaderWay(1)
	way.SetTag("highway", "secondary")
	way.SetTag("railway", "rail")
	assert.True(t, parser.GetAccess(way).IsWay())

	way.ClearTags()
	way.SetTag("highway", "path")
	way.SetTag("railway", "abandoned")
	assert.True(t, parser.GetAccess(way).CanSkip())

	way.SetTag("highway", "track")
	assert.True(t, parser.GetAccess(way).IsWay())

	way.SetTag("highway", "primary")
	way.SetTag("railway", "historic")
	assert.True(t, parser.GetAccess(way).IsWay())

	way.SetTag("motorcar", "no")
	assert.True(t, parser.GetAccess(way).CanSkip())

	way = reader.NewReaderWay(1)
	way.SetTag("highway", "secondary")
	way.SetTag("railway", "tram")
	assert.True(t, parser.GetAccess(way).IsWay())
}

func TestNonHighwaysFallbackSpeed(t *testing.T) {
	em := createCarEM()
	speedParser := NewCarAverageSpeedParser(em)
	avSpeedEnc := speedParser.GetAverageSpeedEnc()
	edgeID := 0

	way := reader.NewReaderWay(1)
	way.SetTag("man_made", "pier")
	edgeIntAccess := ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	speedParser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.InDelta(t, 10, avSpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 1)

	way.ClearTags()
	way.SetTag("railway", "platform")
	edgeIntAccess = ev.NewArrayEdgeIntAccessFromBytes(em.BytesForFlags)
	speedParser.HandleWayTags(edgeID, edgeIntAccess, way, nil)
	assert.InDelta(t, 10, avSpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 1)
}

func TestPedestrianAccess(t *testing.T) {
	em := createCarEM()
	parser := NewCarAccessParser(em, true, true)

	way := reader.NewReaderWay(1)
	way.SetTag("highway", "pedestrian")
	assert.True(t, parser.GetAccess(way).CanSkip())

	way.ClearTags()
	way.SetTag("highway", "pedestrian")
	way.SetTag("motor_vehicle", "no")
	assert.True(t, parser.GetAccess(way).CanSkip())

	way.ClearTags()
	way.SetTag("highway", "pedestrian")
	way.SetTag("motor_vehicle", "destination")
	assert.True(t, parser.GetAccess(way).IsWay())

	way.ClearTags()
	way.SetTag("highway", "pedestrian")
	way.SetTag("motorcar", "no")
	way.SetTag("motor_vehicle", "destination")
	assert.True(t, parser.GetAccess(way).CanSkip())

	way.ClearTags()
	way.SetTag("highway", "pedestrian")
	way.SetTag("motor_vehicle:conditional", "destination @ ( 8:00 - 10:00 )")
	assert.True(t, parser.GetAccess(way).IsWay())
}
