package parsers

import (
	"fmt"

	"gohopper/core/reader"
	"gohopper/core/routing/ev"
	"gohopper/core/storage"
)

// AbstractAverageSpeedParser is the base for vehicle speed parsers.
type AbstractAverageSpeedParser struct {
	avgSpeedEnc   ev.DecimalEncodedValue
	ferrySpeedEnc ev.DecimalEncodedValue
}

func newAbstractAverageSpeedParser(avgSpeedEnc, ferrySpeedEnc ev.DecimalEncodedValue) *AbstractAverageSpeedParser {
	return &AbstractAverageSpeedParser{
		avgSpeedEnc:   avgSpeedEnc,
		ferrySpeedEnc: ferrySpeedEnc,
	}
}

func (p *AbstractAverageSpeedParser) GetAverageSpeedEnc() ev.DecimalEncodedValue {
	return p.avgSpeedEnc
}

func (p *AbstractAverageSpeedParser) HandleWayTags(_ int, _ ev.EdgeIntAccess, _ *reader.ReaderWay, _ *storage.IntsRef) {
	panic("AbstractAverageSpeedParser.HandleWayTags should not be called directly")
}

// SetSpeed sets the speed, clamping to the smallest non-zero value if needed.
func (p *AbstractAverageSpeedParser) SetSpeed(reverse bool, edgeID int, edgeIntAccess ev.EdgeIntAccess, speed float64) {
	min := p.avgSpeedEnc.GetSmallestNonZeroValue()
	if speed < min/2 {
		panic(fmt.Sprintf("speed was %v but cannot be lower than %v/2", speed, min))
	}
	if speed < min {
		speed = min
	}
	p.avgSpeedEnc.SetDecimal(reverse, edgeID, edgeIntAccess, speed)
}
