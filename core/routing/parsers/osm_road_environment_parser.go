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
	env := roadEnvironment(way)
	p.roadEnvironmentEnc.SetEnum(false, edgeID, edgeIntAccess, env)
}

func roadEnvironment(way *reader.ReaderWay) ev.RoadEnvironment {
	route := way.GetTag("route")
	if route == "ferry" || route == "shuttle_train" {
		return ev.RoadEnvironmentFerry
	}
	if way.HasTag("bridge", "yes") {
		return ev.RoadEnvironmentBridge
	}
	if way.HasTag("tunnel", "yes") {
		return ev.RoadEnvironmentTunnel
	}
	if way.HasTag("ford", "yes") || way.HasTag("highway", "ford") {
		return ev.RoadEnvironmentFord
	}
	return ev.RoadEnvironmentRoad
}
