package parsers

import (
	"testing"

	"gohopper/core/reader"
	"gohopper/core/routing/ev"
	"gohopper/core/storage"

	"github.com/stretchr/testify/assert"
)

func TestCountryRule(t *testing.T) {
	maxSpeedEnc := ev.MaxSpeedCreate()
	maxSpeedEnc.Init(ev.NewInitializerConfig())
	parser := NewOSMMaxSpeedParser(maxSpeedEnc)
	relFlags := storage.NewIntsRef(2)

	way := reader.NewReaderWay(29)
	way.SetTag("highway", "primary")
	edgeIntAccess := ev.NewArrayEdgeIntAccess(1)
	edgeID := 0
	way.SetTag("maxspeed", "30")
	parser.HandleWayTags(edgeID, edgeIntAccess, way, relFlags)
	assert.InDelta(t, 30, maxSpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 0.1)

	// Different direction.
	edgeIntAccess = ev.NewArrayEdgeIntAccess(1)
	way = reader.NewReaderWay(29)
	way.SetTag("highway", "primary")
	way.SetTag("maxspeed:forward", "30")
	way.SetTag("maxspeed:backward", "40")
	parser.HandleWayTags(edgeID, edgeIntAccess, way, relFlags)
	assert.InDelta(t, 30, maxSpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 0.1)
	assert.InDelta(t, 40, maxSpeedEnc.GetDecimal(true, edgeID, edgeIntAccess), 0.1)
}

func TestParseMaxSpeed(t *testing.T) {
	way := reader.NewReaderWay(12)
	way.SetTag("maxspeed", "90")
	assert.InDelta(t, 90, ParseMaxSpeed(way, false), 0.01)

	way = reader.NewReaderWay(12)
	way.SetTag("maxspeed", "90")
	way.SetTag("maxspeed:backward", "50")
	assert.InDelta(t, 90, ParseMaxSpeed(way, false), 0.01)
	assert.InDelta(t, 50, ParseMaxSpeed(way, true), 0.01)

	way = reader.NewReaderWay(12)
	way.SetTag("maxspeed", "none")
	assert.InDelta(t, ev.MaxSpeedMissing, ParseMaxSpeed(way, false), 0.01)

	way = reader.NewReaderWay(12)
	way.SetTag("maxspeed", "none")
	way.SetTag("highway", "secondary")
	assert.InDelta(t, ev.MaxSpeedMissing, ParseMaxSpeed(way, false), 0.01)

	way = reader.NewReaderWay(12)
	way.SetTag("maxspeed", "none")
	way.SetTag("highway", "motorway")
	assert.InDelta(t, ev.MaxSpeed150, ParseMaxSpeed(way, false), 0.01)

	// Low maxspeed treated as missing.
	way = reader.NewReaderWay(12)
	way.SetTag("maxspeed", "3")
	assert.InDelta(t, ev.MaxSpeedMissing, ParseMaxSpeed(way, false), 0.01)

	way = reader.NewReaderWay(12)
	way.SetTag("maxspeed", "5")
	assert.InDelta(t, 5, ParseMaxSpeed(way, false), 0.01)

	way = reader.NewReaderWay(12)
	way.SetTag("maxspeed", "3mph")
	assert.InDelta(t, 4.83, ParseMaxSpeed(way, false), 0.01)
}

func TestMaxSpeedNone(t *testing.T) {
	highways := []string{"motorway", "motorway_link", "trunk", "trunk_link", "primary"}
	for _, highway := range highways {
		maxSpeedEnc := ev.MaxSpeedCreate()
		maxSpeedEnc.Init(ev.NewInitializerConfig())
		parser := NewOSMMaxSpeedParser(maxSpeedEnc)
		relFlags := storage.NewIntsRef(2)
		edgeIntAccess := ev.NewArrayEdgeIntAccess(1)
		edgeID := 0
		assert.InDelta(t, 0, maxSpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 0.1)

		way := reader.NewReaderWay(29)
		way.SetTag("highway", highway)
		way.SetTag("maxspeed", "none")
		parser.HandleWayTags(edgeID, edgeIntAccess, way, relFlags)
		assert.InDelta(t, ev.MaxSpeed150, maxSpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 0.1, "highway=%s", highway)
	}
}

func TestSmallMaxSpeed(t *testing.T) {
	maxSpeedEnc := ev.MaxSpeedCreate()
	maxSpeedEnc.Init(ev.NewInitializerConfig())
	parser := NewOSMMaxSpeedParser(maxSpeedEnc)
	relFlags := storage.NewIntsRef(2)
	edgeIntAccess := ev.NewArrayEdgeIntAccess(1)
	edgeID := 0

	way := reader.NewReaderWay(29)
	way.SetTag("highway", "service")
	way.SetTag("maxspeed", "3 mph")
	parser.HandleWayTags(edgeID, edgeIntAccess, way, relFlags)
	assert.InDelta(t, 4, maxSpeedEnc.GetDecimal(false, edgeID, edgeIntAccess), 0.1)
}

func TestParseMaxspeedString(t *testing.T) {
	assert.InDelta(t, 40, ParseMaxspeedString("40 km/h"), 0.1)
	assert.InDelta(t, 40, ParseMaxspeedString("40km/h"), 0.1)
	assert.InDelta(t, 40, ParseMaxspeedString("40kmh"), 0.1)
	assert.InDelta(t, 64.4, ParseMaxspeedString("40mph"), 0.1)
	assert.InDelta(t, 48.3, ParseMaxspeedString("30 mph"), 0.1)
	assert.InDelta(t, 18.5, ParseMaxspeedString("10 knots"), 0.1)
	assert.InDelta(t, 19, ParseMaxspeedString("19 kph"), 0.1)
	assert.InDelta(t, 19, ParseMaxspeedString("19kph"), 0.1)
	assert.InDelta(t, 100, ParseMaxspeedString("100"), 0.1)
	assert.InDelta(t, 100.5, ParseMaxspeedString("100.5"), 0.1)
	assert.InDelta(t, 4.8, ParseMaxspeedString("3 mph"), 0.1)

	assert.InDelta(t, MaxSpeedNone, ParseMaxspeedString("none"), 0.1)
}

func TestParseMaxspeedStringInvalid(t *testing.T) {
	assert.Equal(t, ev.MaxSpeedMissing, ParseMaxspeedString(""))
	assert.Equal(t, ev.MaxSpeedMissing, ParseMaxspeedString("-20"))
	assert.Equal(t, ev.MaxSpeedMissing, ParseMaxspeedString("0"))
	assert.Equal(t, ev.MaxSpeedMissing, ParseMaxspeedString("1"))
	assert.Equal(t, ev.MaxSpeedMissing, ParseMaxspeedString("1km/h"))
	assert.Equal(t, ev.MaxSpeedMissing, ParseMaxspeedString("1mph"))
	assert.Equal(t, ev.MaxSpeedMissing, ParseMaxspeedString("2"))
	assert.Equal(t, ev.MaxSpeedMissing, ParseMaxspeedString("3"))
	assert.Equal(t, ev.MaxSpeedMissing, ParseMaxspeedString("4"))
}
