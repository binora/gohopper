package parsers

import (
	"gohopper/core/reader"
	"gohopper/core/routing/ev"
	"gohopper/core/storage"
)

// OSMRoundaboutParser sets the roundabout boolean encoded value
// based on the junction tag.
type OSMRoundaboutParser struct {
	roundaboutEnc ev.BooleanEncodedValue
}

func NewOSMRoundaboutParser(roundaboutEnc ev.BooleanEncodedValue) *OSMRoundaboutParser {
	return &OSMRoundaboutParser{roundaboutEnc: roundaboutEnc}
}

func (p *OSMRoundaboutParser) HandleWayTags(edgeID int, edgeIntAccess ev.EdgeIntAccess, way *reader.ReaderWay, _ *storage.IntsRef) {
	junction := way.GetTag("junction")
	if junction == "roundabout" || junction == "circular" {
		p.roundaboutEnc.SetBool(false, edgeID, edgeIntAccess, true)
	}
}
