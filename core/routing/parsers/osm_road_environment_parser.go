package parsers

import (
	"gohopper/core/reader"
	"gohopper/core/routing/ev"
	"gohopper/core/storage"
)

// OSMRoadEnvironmentParser classifies edges by their environment
// (ferry, bridge, tunnel, ford, or road).
type OSMRoadEnvironmentParser struct {
	roadEnvironmentEnc *ev.EnumEncodedValue[ev.RoadEnvironment]
}

func NewOSMRoadEnvironmentParser(roadEnvironmentEnc *ev.EnumEncodedValue[ev.RoadEnvironment]) *OSMRoadEnvironmentParser {
	return &OSMRoadEnvironmentParser{roadEnvironmentEnc: roadEnvironmentEnc}
}

func (p *OSMRoadEnvironmentParser) HandleWayTags(edgeID int, edgeIntAccess ev.EdgeIntAccess, way *reader.ReaderWay, _ *storage.IntsRef) {
	re := ev.RoadEnvironmentRoad
	if isFerryRoute(way) {
		re = ev.RoadEnvironmentFerry
	} else if way.HasTag("bridge", "yes") {
		re = ev.RoadEnvironmentBridge
	} else if way.HasTag("tunnel", "yes") {
		re = ev.RoadEnvironmentTunnel
	} else if way.HasTag("ford", "yes") || way.HasTag("highway", "ford") {
		re = ev.RoadEnvironmentFord
	}
	p.roadEnvironmentEnc.SetEnum(false, edgeID, edgeIntAccess, re)
}

// isFerryRoute returns true if the way is a ferry route.
func isFerryRoute(way *reader.ReaderWay) bool {
	route := way.GetTag("route")
	return route == "ferry" || route == "shuttle_train"
}
