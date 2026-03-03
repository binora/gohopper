package parsers

import (
	"strings"

	"gohopper/core/reader"
	"gohopper/core/routing/ev"
	"gohopper/core/storage"
)

// OSMRoadClassLinkParser sets the road_class_link boolean if the highway
// tag ends with "_link".
type OSMRoadClassLinkParser struct {
	roadClassLinkEnc ev.BooleanEncodedValue
}

func NewOSMRoadClassLinkParser(roadClassLinkEnc ev.BooleanEncodedValue) *OSMRoadClassLinkParser {
	return &OSMRoadClassLinkParser{roadClassLinkEnc: roadClassLinkEnc}
}

func (p *OSMRoadClassLinkParser) HandleWayTags(edgeID int, edgeIntAccess ev.EdgeIntAccess, way *reader.ReaderWay, _ *storage.IntsRef) {
	highway := way.GetTag("highway")
	if strings.HasSuffix(highway, "_link") {
		p.roadClassLinkEnc.SetBool(false, edgeID, edgeIntAccess, true)
	}
}
