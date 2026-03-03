package parsers

import (
	"fmt"

	"gohopper/core/reader"
	"gohopper/core/routing/ev"
	"gohopper/core/storage"
)

// OSMWayIDParser stores the OSM way ID in an IntEncodedValue.
type OSMWayIDParser struct {
	wayIDEnc ev.IntEncodedValue
}

func NewOSMWayIDParser(wayIDEnc ev.IntEncodedValue) *OSMWayIDParser {
	return &OSMWayIDParser{wayIDEnc: wayIDEnc}
}

func (p *OSMWayIDParser) HandleWayTags(edgeID int, edgeIntAccess ev.EdgeIntAccess, way *reader.ReaderWay, _ *storage.IntsRef) {
	osmWayID := way.GetID()
	if osmWayID > int64(p.wayIDEnc.GetMaxStorableInt()) {
		panic(fmt.Sprintf("Cannot store OSM way ID %d because the osm_way_id encoded value only allows values up to %d.",
			osmWayID, p.wayIDEnc.GetMaxStorableInt()))
	}
	p.wayIDEnc.SetInt(false, edgeID, edgeIntAccess, int32(osmWayID))
}
