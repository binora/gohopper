package parsers

import (
	"math"

	"gohopper/core/reader"
	"gohopper/core/routing/ev"
	"gohopper/core/storage"
)

var defaultSpeedMap = map[string]float64{
	"motorway":       100,
	"motorway_link":  70,
	"trunk":          70,
	"trunk_link":     65,
	"primary":        65,
	"primary_link":   60,
	"secondary":      60,
	"secondary_link": 50,
	"tertiary":       50,
	"tertiary_link":  40,
	"unclassified":   30,
	"residential":    30,
	"living_street":  6,
	"pedestrian":     6,
	"service":        20,
	"road":           20,
	"track":          15,
}

var trackTypeSpeedMap = map[string]float64{
	"grade1": 20,
	"grade2": 15,
	"grade3": 10,
}

var badSurfaceSet = map[string]bool{
	"cobblestone":        true,
	"grass_paver":        true,
	"gravel":             true,
	"sand":               true,
	"paving_stones":      true,
	"dirt":               true,
	"ground":             true,
	"grass":              true,
	"unpaved":            true,
	"compacted":          true,
	"wood":               true,
	"pebblestone":        true,
	"fine_gravel":        true,
	"earth":              true,
	"sett":               true,
	"unhewn_cobblestone": true,
}

const badSurfaceSpeed = 30.0

// CarAverageSpeedParser computes the average speed for car on a way.
type CarAverageSpeedParser struct {
	*AbstractAverageSpeedParser
}

func NewCarAverageSpeedParser(lookup ev.EncodedValueLookup) *CarAverageSpeedParser {
	avgSpeedEnc := lookup.GetDecimalEncodedValue(ev.VehicleSpeedKey("car"))
	ferrySpeedEnc := lookup.GetDecimalEncodedValue(ev.FerrySpeedKey)
	return &CarAverageSpeedParser{
		AbstractAverageSpeedParser: newAbstractAverageSpeedParser(avgSpeedEnc, ferrySpeedEnc),
	}
}

func (p *CarAverageSpeedParser) HandleWayTags(edgeID int, edgeIntAccess ev.EdgeIntAccess, way *reader.ReaderWay, _ *storage.IntsRef) {
	if IsFerry(way) {
		ferrySpeed := p.ferrySpeedEnc.GetDecimal(false, edgeID, edgeIntAccess)
		if ferrySpeed == 0 {
			ferrySpeed = GetFerrySpeed(way)
			ferrySpeed = MinMax(ferrySpeed, p.ferrySpeedEnc)
		}
		p.SetSpeed(false, edgeID, edgeIntAccess, ferrySpeed)
		p.SetSpeed(true, edgeID, edgeIntAccess, ferrySpeed)
		return
	}

	speed := p.getSpeed(way)
	speed = p.ApplyBadSurfaceSpeed(way, speed)

	p.setSpeedWithMaxspeed(false, edgeID, edgeIntAccess, way, speed)
	p.setSpeedWithMaxspeed(true, edgeID, edgeIntAccess, way, speed)
}

func (p *CarAverageSpeedParser) getSpeed(way *reader.ReaderWay) float64 {
	highway := way.GetTag("highway")

	// Track type overrides default speed for tracks.
	if highway == "track" {
		trackType := way.GetTag("tracktype")
		if speed, ok := trackTypeSpeedMap[trackType]; ok {
			return speed
		}
	}

	if speed, ok := defaultSpeedMap[highway]; ok {
		return speed
	}

	// Fallback for non-highway ways (man_made=pier, railway=platform, etc.)
	return 10
}

func (p *CarAverageSpeedParser) setSpeedWithMaxspeed(reverse bool, edgeID int, edgeIntAccess ev.EdgeIntAccess, way *reader.ReaderWay, defaultSpeed float64) {
	speed := p.applyMaxSpeed(way, defaultSpeed, reverse)
	p.SetSpeed(reverse, edgeID, edgeIntAccess, speed)
}

func (p *CarAverageSpeedParser) applyMaxSpeed(way *reader.ReaderWay, speed float64, reverse bool) float64 {
	maxSpeed := ParseMaxSpeed(way, reverse)
	if maxSpeed == ev.MaxSpeedMissing {
		return speed
	}
	// Use 90% of the max speed as the average.
	return math.Min(maxSpeed*0.9, p.avgSpeedEnc.GetMaxStorableDecimal())
}

// ApplyBadSurfaceSpeed caps the speed if the surface is a known bad surface.
func (p *CarAverageSpeedParser) ApplyBadSurfaceSpeed(way *reader.ReaderWay, speed float64) float64 {
	surface := way.GetTag("surface")
	if surface == "" {
		return speed
	}
	if badSurfaceSet[surface] && speed > badSurfaceSpeed {
		return badSurfaceSpeed
	}
	return speed
}
