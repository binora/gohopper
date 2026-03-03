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

// CarAverageSpeedParser computes the average speed for a car on a way.
type CarAverageSpeedParser struct {
	*AbstractAverageSpeedParser
}

func NewCarAverageSpeedParser(lookup ev.EncodedValueLookup) *CarAverageSpeedParser {
	return &CarAverageSpeedParser{
		AbstractAverageSpeedParser: newAbstractAverageSpeedParser(
			lookup.GetDecimalEncodedValue(ev.VehicleSpeedKey("car")),
			lookup.GetDecimalEncodedValue(ev.FerrySpeedKey),
		),
	}
}

func (p *CarAverageSpeedParser) HandleWayTags(edgeID int, edgeIntAccess ev.EdgeIntAccess, way *reader.ReaderWay, _ *storage.IntsRef) {
	if IsFerry(way) {
		p.handleFerrySpeed(edgeID, edgeIntAccess, way)
		return
	}

	speed := p.ApplyBadSurfaceSpeed(way, p.getSpeed(way))
	p.setSpeedWithMaxspeed(false, edgeID, edgeIntAccess, way, speed)
	p.setSpeedWithMaxspeed(true, edgeID, edgeIntAccess, way, speed)
}

func (p *CarAverageSpeedParser) handleFerrySpeed(edgeID int, edgeIntAccess ev.EdgeIntAccess, way *reader.ReaderWay) {
	speed := p.ferrySpeedEnc.GetDecimal(false, edgeID, edgeIntAccess)
	if speed == 0 {
		speed = MinMax(GetFerrySpeed(way), p.ferrySpeedEnc)
	}
	p.SetSpeed(false, edgeID, edgeIntAccess, speed)
	p.SetSpeed(true, edgeID, edgeIntAccess, speed)
}

func (p *CarAverageSpeedParser) getSpeed(way *reader.ReaderWay) float64 {
	highway := way.GetTag("highway")

	if highway == "track" {
		if speed, ok := trackTypeSpeedMap[way.GetTag("tracktype")]; ok {
			return speed
		}
	}

	if speed, ok := defaultSpeedMap[highway]; ok {
		return speed
	}

	return 10
}

func (p *CarAverageSpeedParser) setSpeedWithMaxspeed(reverse bool, edgeID int, edgeIntAccess ev.EdgeIntAccess, way *reader.ReaderWay, speed float64) {
	maxSpeed := ParseMaxSpeed(way, reverse)
	if maxSpeed != ev.MaxSpeedMissing {
		speed = math.Min(maxSpeed*0.9, p.avgSpeedEnc.GetMaxStorableDecimal())
	}
	p.SetSpeed(reverse, edgeID, edgeIntAccess, speed)
}

// ApplyBadSurfaceSpeed caps the speed if the surface is a known bad surface.
func (p *CarAverageSpeedParser) ApplyBadSurfaceSpeed(way *reader.ReaderWay, speed float64) float64 {
	if surface := way.GetTag("surface"); badSurfaceSet[surface] && speed > badSurfaceSpeed {
		return badSurfaceSpeed
	}
	return speed
}
