package parsers

import (
	"strings"

	"gohopper/core/reader"
	"gohopper/core/routing/ev"
	"gohopper/core/routing/util"
	"gohopper/core/storage"
)

var carHighwayValues = map[string]bool{
	"motorway":      true,
	"motorway_link": true,
	"trunk":         true,
	"trunk_link":    true,
	"primary":       true,
	"primary_link":  true,
	"secondary":     true,
	"secondary_link": true,
	"tertiary":      true,
	"tertiary_link": true,
	"unclassified":  true,
	"residential":   true,
	"living_street": true,
	"service":       true,
	"road":          true,
	"track":         true,
	"pedestrian":    true,
}

var carTrackTypeValues = map[string]bool{
	"grade1": true,
	"grade2": true,
	"grade3": true,
	"":       true,
}

var carBarriers = map[string]bool{
	"kissing_gate":      true,
	"fence":             true,
	"bollard":           true,
	"stile":             true,
	"turnstile":         true,
	"cycle_barrier":     true,
	"motorcycle_barrier": true,
	"block":             true,
	"bus_trap":          true,
	"sump_buster":       true,
	"jersey_barrier":    true,
}

// CarAccessParser determines whether a car can use a way.
type CarAccessParser struct {
	*AbstractAccessParser
}

// NewCarAccessParser creates a new car access parser.
func NewCarAccessParser(lookup ev.EncodedValueLookup, blockFords bool, blockPrivate bool) *CarAccessParser {
	accessEnc := lookup.GetBooleanEncodedValue(ev.VehicleAccessKey("car"))
	roundaboutEnc := lookup.GetBooleanEncodedValue(ev.RoundaboutKey)

	restrictionKeys := ToOSMRestrictions(util.TransportationModeCar)
	base := newAbstractAccessParser(accessEnc, roundaboutEnc, restrictionKeys)

	// Additional restricted values for car.
	base.restrictedValues["agricultural"] = true
	base.restrictedValues["forestry"] = true
	base.restrictedValues["delivery"] = true

	// Car-specific barriers.
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

	// Check pedestrian-specific handling.
	if highway == "pedestrian" {
		return p.handlePedestrianAccess(way)
	}

	access := p.getAccess(way, carHighwayValues, carTrackTypeValues)
	if access.CanSkip() {
		return access
	}

	// Exclude footway/cycleway/steps even if motor_vehicle=yes for specific tags.
	if highway == "footway" || highway == "cycleway" || highway == "steps" {
		return util.WayAccessCanSkip
	}

	return access
}

func (p *CarAccessParser) handlePedestrianAccess(way *reader.ReaderWay) util.WayAccess {
	// For pedestrian highways, only allow if explicit motor_vehicle access is allowed.
	for _, key := range p.restrictionKeys {
		val := way.GetTag(key)
		if val != "" {
			if p.allowedValues[val] {
				return util.WayAccessWay
			}
			if p.restrictedValues[val] {
				return util.WayAccessCanSkip
			}
		}
		// Check conditional
		condVal := way.GetTag(key + ":conditional")
		if condVal != "" {
			idx := strings.Index(condVal, "@")
			if idx >= 0 {
				accessPart := strings.TrimSpace(condVal[:idx])
				if p.allowedValues[accessPart] {
					return util.WayAccessWay
				}
			}
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

	if !access.IsFerry() {
		// Check oneway tags.
		if isOneway(way) {
			p.accessEnc.SetBool(false, edgeID, edgeIntAccess, true)
		} else if isReverseOneway(way) {
			p.accessEnc.SetBool(true, edgeID, edgeIntAccess, true)
		} else if p.isForwardBlocked(way) {
			p.accessEnc.SetBool(true, edgeID, edgeIntAccess, true)
		} else if p.isBackwardBlocked(way) {
			p.accessEnc.SetBool(false, edgeID, edgeIntAccess, true)
		} else {
			p.accessEnc.SetBool(false, edgeID, edgeIntAccess, true)
			p.accessEnc.SetBool(true, edgeID, edgeIntAccess, true)
		}

		// Check roundabout: force oneway.
		if p.roundaboutEnc.GetBool(false, edgeID, edgeIntAccess) {
			p.accessEnc.SetBool(true, edgeID, edgeIntAccess, false)
		}
	} else {
		// Ferry: bidirectional.
		p.accessEnc.SetBool(false, edgeID, edgeIntAccess, true)
		p.accessEnc.SetBool(true, edgeID, edgeIntAccess, true)
	}
}

func (p *CarAccessParser) isForwardBlocked(way *reader.ReaderWay) bool {
	for _, prefix := range []string{"vehicle", "motor_vehicle", "motorcar"} {
		if way.HasTag(prefix+":forward", "no") {
			return true
		}
	}
	return false
}

func (p *CarAccessParser) isBackwardBlocked(way *reader.ReaderWay) bool {
	for _, prefix := range []string{"vehicle", "motor_vehicle", "motorcar"} {
		if way.HasTag(prefix+":backward", "no") {
			return true
		}
	}
	return false
}
