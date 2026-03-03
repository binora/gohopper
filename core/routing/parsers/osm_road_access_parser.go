package parsers

import (
	"strings"

	"gohopper/core/reader"
	"gohopper/core/routing/ev"
	"gohopper/core/routing/util"
	"gohopper/core/storage"
)

// OSMRoadAccessParser parses restriction tags to determine road access levels.
type OSMRoadAccessParser struct {
	roadAccessEnc  *ev.EnumEncodedValue[ev.RoadAccess]
	restrictionKeys []string
}

func NewOSMRoadAccessParser(roadAccessEnc *ev.EnumEncodedValue[ev.RoadAccess], restrictionKeys []string) *OSMRoadAccessParser {
	return &OSMRoadAccessParser{
		roadAccessEnc:  roadAccessEnc,
		restrictionKeys: restrictionKeys,
	}
}

// ToOSMRestrictions returns the OSM tag keys for access restrictions
// for the given transportation mode.
func ToOSMRestrictions(mode util.TransportationMode) []string {
	switch mode {
	case util.TransportationModeCar:
		return []string{"motorcar", "motor_vehicle", "vehicle", "access"}
	case util.TransportationModeBike:
		return []string{"bicycle", "vehicle", "access"}
	case util.TransportationModeFoot:
		return []string{"foot", "access"}
	case util.TransportationModeMotorcycle:
		return []string{"motorcycle", "motor_vehicle", "vehicle", "access"}
	case util.TransportationModeHGV:
		return []string{"hgv", "motor_vehicle", "vehicle", "access"}
	default:
		return []string{"access"}
	}
}

func (p *OSMRoadAccessParser) HandleWayTags(edgeID int, edgeIntAccess ev.EdgeIntAccess, way *reader.ReaderWay, _ *storage.IntsRef) {
	var bestAccess ev.RoadAccess
	found := false

	for _, key := range p.restrictionKeys {
		val := way.GetTag(key)
		if val == "" {
			continue
		}
		// Handle semicolon-separated values — pick the least restrictive
		parts := strings.Split(val, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			ra := ev.RoadAccessFind(part)
			if !found || ra < bestAccess {
				bestAccess = ra
				found = true
			}
		}
	}

	if found && bestAccess != ev.RoadAccessYes {
		p.roadAccessEnc.SetEnum(false, edgeID, edgeIntAccess, bestAccess)
	}
}
