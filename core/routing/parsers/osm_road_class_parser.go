package parsers

import (
	"strings"

	"gohopper/core/reader"
	"gohopper/core/routing/ev"
	"gohopper/core/storage"
)

// OSMRoadClassParser sets the road_class encoded value from the highway tag.
type OSMRoadClassParser struct {
	roadClassEnc *ev.EnumEncodedValue[ev.RoadClass]
}

func NewOSMRoadClassParser(roadClassEnc *ev.EnumEncodedValue[ev.RoadClass]) *OSMRoadClassParser {
	return &OSMRoadClassParser{roadClassEnc: roadClassEnc}
}

func (p *OSMRoadClassParser) HandleWayTags(edgeID int, edgeIntAccess ev.EdgeIntAccess, way *reader.ReaderWay, _ *storage.IntsRef) {
	highway := way.GetTag("highway")
	rc := ev.RoadClassFind(highway)
	if rc == ev.RoadClassOther && strings.HasSuffix(highway, "_link") {
		rc = ev.RoadClassFind(strings.TrimSuffix(highway, "_link"))
	}
	if rc != ev.RoadClassOther {
		p.roadClassEnc.SetEnum(false, edgeID, edgeIntAccess, rc)
	}
}
