package parsers

import (
	"strings"

	"gohopper/core/reader"
	"gohopper/core/routing/ev"
	"gohopper/core/routing/util"
	"gohopper/core/storage"
)

var carHighwayValues = map[string]bool{
	"motorway":       true,
	"motorway_link":  true,
	"trunk":          true,
	"trunk_link":     true,
	"primary":        true,
	"primary_link":   true,
	"secondary":      true,
	"secondary_link": true,
	"tertiary":       true,
	"tertiary_link":  true,
	"unclassified":   true,
	"residential":    true,
	"living_street":  true,
	"service":        true,
	"road":           true,
	"track":          true,
	"pedestrian":     true,
}

var carTrackTypeValues = map[string]bool{
	"grade1": true,
	"grade2": true,
	"grade3": true,
	"":       true,
}

var carBarriers = map[string]bool{
	"kissing_gate":       true,
	"fence":              true,
	"bollard":            true,
	"stile":              true,
	"turnstile":          true,
	"cycle_barrier":      true,
	"motorcycle_barrier": true,
	"block":              true,
	"bus_trap":           true,
	"sump_buster":        true,
	"jersey_barrier":     true,
}

// directionPrefixes are the OSM key prefixes used for directional access checks.
var directionPrefixes = []string{"vehicle", "motor_vehicle", "motorcar"}

// CarAccessParser determines whether a car can use a way.
type CarAccessParser struct {
	*AbstractAccessParser
}

// NewCarAccessParser creates a new car access parser.
func NewCarAccessParser(lookup ev.EncodedValueLookup, blockFords, blockPrivate bool) *CarAccessParser {
	accessEnc := lookup.GetBooleanEncodedValue(ev.VehicleAccessKey("car"))
	roundaboutEnc := lookup.GetBooleanEncodedValue(ev.RoundaboutKey)

	base := newAbstractAccessParser(accessEnc, roundaboutEnc, ToOSMRestrictions(util.TransportationModeCar))

	base.restrictedValues["agricultural"] = true
	base.restrictedValues["forestry"] = true
	base.restrictedValues["delivery"] = true

	for k, v := range carBarriers {
		base.barriers[k] = v
	}

	base.blockFords = blockFords
	base.blockPrivate = blockPrivate
	if !blockPrivate {
		delete(base.restrictedValues, "private")
		delete(base.restrictedValues, "permit")
	}

	return &CarAccessParser{AbstractAccessParser: base}
}

// GetAccess returns the access type for the way.
func (p *CarAccessParser) GetAccess(way *reader.ReaderWay) util.WayAccess {
	highway := way.GetTag("highway")

	if highway == "pedestrian" {
		return p.pedestrianAccess(way)
	}

	access := p.getAccess(way, carHighwayValues, carTrackTypeValues)
	if access.CanSkip() {
		return access
	}

	if highway == "footway" || highway == "cycleway" || highway == "steps" {
		return util.WayAccessCanSkip
	}

	return access
}

// pedestrianAccess checks whether motor vehicle access is explicitly allowed
// on a pedestrian highway, including conditional tags.
func (p *CarAccessParser) pedestrianAccess(way *reader.ReaderWay) util.WayAccess {
	for _, key := range p.restrictionKeys {
		if val := way.GetTag(key); val != "" {
			if p.allowedValues[val] {
				return util.WayAccessWay
			}
			if p.restrictedValues[val] {
				return util.WayAccessCanSkip
			}
		}

		condVal := way.GetTag(key + ":conditional")
		if condVal == "" {
			continue
		}
		idx := strings.Index(condVal, "@")
		if idx >= 0 && p.allowedValues[strings.TrimSpace(condVal[:idx])] {
			return util.WayAccessWay
		}
	}
	return util.WayAccessCanSkip
}

// HandleWayTags implements TagParser, setting forward/backward access booleans.
func (p *CarAccessParser) HandleWayTags(edgeID int, edgeIntAccess ev.EdgeIntAccess, way *reader.ReaderWay, _ *storage.IntsRef) {
	access := p.GetAccess(way)
	if access.CanSkip() {
		return
	}

	if access.IsFerry() {
		p.accessEnc.SetBool(false, edgeID, edgeIntAccess, true)
		p.accessEnc.SetBool(true, edgeID, edgeIntAccess, true)
		return
	}

	if isOneway(way) {
		p.accessEnc.SetBool(false, edgeID, edgeIntAccess, true)
	} else if isReverseOneway(way) {
		p.accessEnc.SetBool(true, edgeID, edgeIntAccess, true)
	} else if isDirectionBlocked(way, "forward") {
		p.accessEnc.SetBool(true, edgeID, edgeIntAccess, true)
	} else if isDirectionBlocked(way, "backward") {
		p.accessEnc.SetBool(false, edgeID, edgeIntAccess, true)
	} else {
		p.accessEnc.SetBool(false, edgeID, edgeIntAccess, true)
		p.accessEnc.SetBool(true, edgeID, edgeIntAccess, true)
	}

	if p.roundaboutEnc.GetBool(false, edgeID, edgeIntAccess) {
		p.accessEnc.SetBool(true, edgeID, edgeIntAccess, false)
	}
}

// isDirectionBlocked returns true if any motor vehicle prefix blocks the given direction.
func isDirectionBlocked(way *reader.ReaderWay, direction string) bool {
	for _, prefix := range directionPrefixes {
		if way.HasTag(prefix+":"+direction, "no") {
			return true
		}
	}
	return false
}
