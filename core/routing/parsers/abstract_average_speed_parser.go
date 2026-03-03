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

func newAbstractAverageSpeedParser(avgSpeedEnc ev.DecimalEncodedValue, ferrySpeedEnc ev.DecimalEncodedValue) *AbstractAverageSpeedParser {
	return &AbstractAverageSpeedParser{
		avgSpeedEnc:   avgSpeedEnc,
		ferrySpeedEnc: ferrySpeedEnc,
	}
}

func (p *AbstractAverageSpeedParser) GetAverageSpeedEnc() ev.DecimalEncodedValue { return p.avgSpeedEnc }

// HandleWayTags should be overridden by the concrete parser.
func (p *AbstractAverageSpeedParser) HandleWayTags(edgeID int, edgeIntAccess ev.EdgeIntAccess, way *reader.ReaderWay, _ *storage.IntsRef) {
	panic("AbstractAverageSpeedParser.HandleWayTags should not be called directly")
}

// SetSpeed sets the speed, clamping to the smallest non-zero value.
func (p *AbstractAverageSpeedParser) SetSpeed(reverse bool, edgeID int, edgeIntAccess ev.EdgeIntAccess, speed float64) {
	smallestNonZero := p.avgSpeedEnc.GetSmallestNonZeroValue()
	if speed < smallestNonZero/2 {
		panic(fmt.Sprintf("Speed was %v but cannot be lower than %v/2", speed, smallestNonZero))
	}
	if speed < smallestNonZero {
		speed = smallestNonZero
	}
	p.avgSpeedEnc.SetDecimal(reverse, edgeID, edgeIntAccess, speed)
}
