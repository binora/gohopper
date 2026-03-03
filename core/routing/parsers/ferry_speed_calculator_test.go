package parsers

import (
	"math"
	"testing"

	"gohopper/core/reader"
	"gohopper/core/routing/ev"
	"gohopper/core/storage"

	"github.com/stretchr/testify/assert"
)

func TestFerrySpeed(t *testing.T) {
	ferrySpeedEnc := ev.FerrySpeedCreate()
	ferrySpeedEnc.Init(ev.NewInitializerConfig())
	calc := NewFerrySpeedCalculator(ferrySpeedEnc)

	way := reader.NewReaderWay(1)
	way.SetTag("route", "ferry")
	way.SetTag("edge_distance", 30000.0)
	way.SetTag("speed_from_duration", 30.0/0.5)

	edgeIntAccess := ev.NewArrayEdgeIntAccess(1)
	edgeID := 0
	calc.HandleWayTags(edgeID, edgeIntAccess, way, storage.EmptyIntsRef)
	assert.InDelta(t, 44, ferrySpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 1)

	// Shuttle train.
	way = reader.NewReaderWay(1)
	way.SetTag("route", "shuttle_train")
	way.SetTag("motorcar", "yes")
	way.SetTag("bicycle", "no")
	way.SetTag("way_distance", 50000.0)
	way.SetTag("speed_from_duration", 50.0/(35.0/60))
	edgeIntAccess = ev.NewArrayEdgeIntAccess(1)
	calc.HandleWayTags(edgeID, edgeIntAccess, way, storage.EmptyIntsRef)
	assert.InDelta(t, 62, ferrySpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 1)

	// Very short and slow ferry.
	way = reader.NewReaderWay(1)
	way.SetTag("route", "ferry")
	way.SetTag("motorcar", "yes")
	way.SetTag("way_distance", 100.0)
	way.SetTag("speed_from_duration", 0.1/(12.0/60))
	edgeIntAccess = ev.NewArrayEdgeIntAccess(1)
	calc.HandleWayTags(edgeID, edgeIntAccess, way, storage.EmptyIntsRef)
	assert.InDelta(t, 2, ferrySpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 1)

	// Missing duration: short ferry.
	way = reader.NewReaderWay(1)
	way.SetTag("route", "ferry")
	way.SetTag("motorcar", "yes")
	way.SetTag("edge_distance", 100.0)
	edgeIntAccess = ev.NewArrayEdgeIntAccess(1)
	calc.HandleWayTags(edgeID, edgeIntAccess, way, storage.EmptyIntsRef)
	assert.InDelta(t, 2, ferrySpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 1)
}

func TestRawSpeed(t *testing.T) {
	ferrySpeedEnc := ev.FerrySpeedCreate()
	ferrySpeedEnc.Init(ev.NewInitializerConfig())

	checkSpeed := func(speedFromDuration *float64, edgeDistance *float64, expected float64) {
		way := reader.NewReaderWay(0)
		if speedFromDuration != nil {
			way.SetTag("speed_from_duration", *speedFromDuration)
		}
		if edgeDistance != nil {
			way.SetTag("edge_distance", *edgeDistance)
		}
		actual := MinMax(GetFerrySpeed(way), ferrySpeedEnc)
		assert.InDelta(t, expected, actual, 0.01)
	}

	s30 := 30.0
	s45 := 45.0
	s100 := 100.0
	s05 := 0.5
	d100 := 100.0
	d1000 := 1000.0

	// speed_from_duration set
	checkSpeed(&s30, nil, math.Round(30.0/1.4))
	checkSpeed(&s45, nil, math.Round(45.0/1.4))
	// Above max (capped)
	checkSpeed(&s100, nil, ferrySpeedEnc.GetMaxStorableDecimal())
	// Below smallest storable
	checkSpeed(&s05, nil, ferrySpeedEnc.GetSmallestNonZeroValue())

	// No speed, but distance
	checkSpeed(nil, &d100, ferrySpeedEnc.GetSmallestNonZeroValue())
	checkSpeed(nil, &d1000, 6)

	// No speed, no distance — panics
	assert.Panics(t, func() {
		checkSpeed(nil, nil, 6)
	})
}
