package parsers

import (
	"strings"

	"gohopper/core/reader"
	"gohopper/core/routing/ev"
	"gohopper/core/routing/util"
	"gohopper/core/storage"
)

// AbstractAccessParser is the base for vehicle access parsers.
type AbstractAccessParser struct {
	accessEnc       ev.BooleanEncodedValue
	roundaboutEnc   ev.BooleanEncodedValue
	restrictionKeys []string
	restrictedValues map[string]bool
	allowedValues    map[string]bool
	barriers         map[string]bool
	blockFords       bool
	blockPrivate     bool
}

func newAbstractAccessParser(accessEnc ev.BooleanEncodedValue, roundaboutEnc ev.BooleanEncodedValue, restrictionKeys []string) *AbstractAccessParser {
	return &AbstractAccessParser{
		accessEnc:       accessEnc,
		roundaboutEnc:   roundaboutEnc,
		restrictionKeys: restrictionKeys,
		restrictedValues: map[string]bool{
			"no":         true,
			"restricted": true,
			"military":   true,
			"emergency":  true,
			"private":    true,
			"permit":     true,
		},
		allowedValues: map[string]bool{
			"yes":         true,
			"designated":  true,
			"official":    true,
			"permissive":  true,
			"destination": true,
		},
		barriers:     make(map[string]bool),
		blockFords:   true,
		blockPrivate: true,
	}
}

func (p *AbstractAccessParser) GetAccessEnc() ev.BooleanEncodedValue { return p.accessEnc }
func (p *AbstractAccessParser) IsBlockFords() bool                   { return p.blockFords }

// HandleWayTags dispatches to the subclass-specific implementation.
// This is the 4-arg version required by TagParser interface.
func (p *AbstractAccessParser) HandleWayTags(edgeID int, edgeIntAccess ev.EdgeIntAccess, way *reader.ReaderWay, _ *storage.IntsRef) {
	// This should be overridden by the concrete parser (CarAccessParser etc.)
	panic("AbstractAccessParser.HandleWayTags should not be called directly")
}

// IsBarrier returns true if the node represents a barrier for this vehicle type.
func (p *AbstractAccessParser) IsBarrier(node *reader.ReaderNode) bool {
	barrier := node.GetTag("barrier")
	if barrier == "" {
		return false
	}

	if p.blockFords && (node.HasTag("ford", "yes") || node.HasTag("ford", "pond")) {
		return true
	}

	if p.barriers[barrier] {
		// Check if there is an explicit access override.
		for _, key := range p.restrictionKeys {
			val := node.GetTag(key)
			if val != "" {
				if p.allowedValues[val] {
					return false
				}
				if p.restrictedValues[val] {
					return true
				}
			}
		}
		// Check generic access/bicycle/foot etc.
		accessVal := node.GetTag("access")
		if accessVal != "" && p.allowedValues[accessVal] {
			return false
		}
		return true
	}
	return false
}

// HandleBarrierEdge sets access for a barrier edge (both directions blocked).
func (p *AbstractAccessParser) HandleBarrierEdge(edgeID int, edgeIntAccess ev.EdgeIntAccess, _ map[string]any) {
	p.accessEnc.SetBool(false, edgeID, edgeIntAccess, false)
	p.accessEnc.SetBool(true, edgeID, edgeIntAccess, false)
}

// isOneway returns true if the way is oneway in the forward direction.
func isOneway(way *reader.ReaderWay) bool {
	ow := way.GetTag("oneway")
	return ow == "yes" || ow == "1" || ow == "true"
}

// isReverseOneway returns true if the way is oneway in the reverse direction.
func isReverseOneway(way *reader.ReaderWay) bool {
	ow := way.GetTag("oneway")
	return ow == "-1" || ow == "reverse"
}

// isValueAllowed checks if at least one semicolon-separated part is in the allowed set.
func isValueAllowed(value string, allowed map[string]bool) bool {
	for _, part := range strings.Split(value, ";") {
		part = strings.TrimSpace(part)
		if allowed[part] {
			return true
		}
	}
	return false
}

// isValueRestricted checks if ALL semicolon-separated parts are in the restricted set.
func isValueRestricted(value string, restricted map[string]bool) bool {
	parts := strings.Split(value, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if !restricted[part] {
			return false
		}
	}
	return true
}

// getAccess determines the WayAccess for a way based on restriction tags.
func (p *AbstractAccessParser) getAccess(way *reader.ReaderWay, highwayValues map[string]bool, trackTypeValues map[string]bool) util.WayAccess {
	highway := way.GetTag("highway")

	if IsFerry(way) {
		// Check for explicit vehicle restriction on ferry.
		for _, key := range p.restrictionKeys {
			val := way.GetTag(key)
			if val != "" {
				if p.allowedValues[val] {
					return util.WayAccessFerry
				}
				if p.restrictedValues[val] {
					return util.WayAccessCanSkip
				}
			}
		}
		// No explicit restriction — check if highway
		if highway != "" {
			return util.WayAccessCanSkip
		}
		return util.WayAccessFerry
	}

	if highway == "" {
		return util.WayAccessCanSkip
	}

	if !highwayValues[highway] {
		return util.WayAccessCanSkip
	}

	if highway == "track" {
		trackType := way.GetTag("tracktype")
		if trackType != "" && !trackTypeValues[trackType] {
			return util.WayAccessCanSkip
		}
	}

	// Check ford
	if p.blockFords && way.HasTag("ford", "yes") {
		// Check if there's an explicit allow for our vehicle.
		for _, key := range p.restrictionKeys {
			val := way.GetTag(key)
			if val != "" && p.allowedValues[val] {
				return util.WayAccessWay
			}
		}
		return util.WayAccessCanSkip
	}

	// Check restriction tags in order of specificity.
	for _, key := range p.restrictionKeys {
		val := way.GetTag(key)
		if val == "" {
			continue
		}
		if isValueAllowed(val, p.allowedValues) {
			return util.WayAccessWay
		}
		if isValueRestricted(val, p.restrictedValues) {
			// Check for temporal conditional override at a higher priority.
			if p.hasConditionalOverride(way, key) {
				return util.WayAccessWay
			}
			return util.WayAccessCanSkip
		}
	}

	// Check service=emergency_access
	if way.GetTag("service") == "emergency_access" {
		return util.WayAccessCanSkip
	}

	return util.WayAccessWay
}

// hasConditionalOverride checks if a more specific conditional tag could override a restriction.
func (p *AbstractAccessParser) hasConditionalOverride(way *reader.ReaderWay, restrictedKey string) bool {
	for _, key := range p.restrictionKeys {
		if key == restrictedKey {
			break
		}
		condVal := way.GetTag(key + ":conditional")
		if condVal == "" {
			continue
		}
		// Extract the access value before the "@"
		idx := strings.Index(condVal, "@")
		if idx < 0 {
			continue
		}
		accessPart := strings.TrimSpace(condVal[:idx])
		if p.allowedValues[accessPart] {
			return true
		}
	}
	return false
}
