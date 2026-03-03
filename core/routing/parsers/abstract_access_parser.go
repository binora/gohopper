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
	accessEnc        ev.BooleanEncodedValue
	roundaboutEnc    ev.BooleanEncodedValue
	restrictionKeys  []string
	restrictedValues map[string]bool
	allowedValues    map[string]bool
	barriers         map[string]bool
	blockFords       bool
	blockPrivate     bool
}

func newAbstractAccessParser(accessEnc, roundaboutEnc ev.BooleanEncodedValue, restrictionKeys []string) *AbstractAccessParser {
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

func (p *AbstractAccessParser) HandleWayTags(_ int, _ ev.EdgeIntAccess, _ *reader.ReaderWay, _ *storage.IntsRef) {
	panic("AbstractAccessParser.HandleWayTags should not be called directly")
}

// IsBarrier returns true if the node represents a barrier for this vehicle type.
func (p *AbstractAccessParser) IsBarrier(node *reader.ReaderNode) bool {
	barrier := node.GetTag("barrier")
	if barrier == "" {
		return p.isFordBlocked(node)
	}

	for _, key := range p.restrictionKeys {
		val := node.GetTag(key)
		if val == "" {
			continue
		}
		if p.restrictedValues[val] {
			return true
		}
		if p.allowedValues[val] {
			return false
		}
	}

	if p.barriers[barrier] {
		return true
	}

	return p.isFordBlocked(node)
}

func (p *AbstractAccessParser) isFordBlocked(node *reader.ReaderNode) bool {
	return p.blockFords && (node.HasTag("ford", "yes") || node.HasTag("ford", "pond"))
}

// HandleBarrierEdge blocks access in both directions for a barrier edge.
func (p *AbstractAccessParser) HandleBarrierEdge(edgeID int, edgeIntAccess ev.EdgeIntAccess, _ map[string]any) {
	p.accessEnc.SetBool(false, edgeID, edgeIntAccess, false)
	p.accessEnc.SetBool(true, edgeID, edgeIntAccess, false)
}

func isOneway(way *reader.ReaderWay) bool {
	ow := way.GetTag("oneway")
	return ow == "yes" || ow == "1" || ow == "true"
}

func isReverseOneway(way *reader.ReaderWay) bool {
	ow := way.GetTag("oneway")
	return ow == "-1" || ow == "reverse"
}

// anySemicolonPartInSet returns true if any semicolon-separated part of value is in the set.
func anySemicolonPartInSet(value string, set map[string]bool) bool {
	for _, part := range strings.Split(value, ";") {
		if set[strings.TrimSpace(part)] {
			return true
		}
	}
	return false
}

// getAccess determines the WayAccess for a way based on restriction tags.
func (p *AbstractAccessParser) getAccess(way *reader.ReaderWay, highwayValues, trackTypeValues map[string]bool) util.WayAccess {
	highway := way.GetTag("highway")

	if IsFerry(way) {
		return p.getFerryAccess(way, highway)
	}

	if highway == "" || !highwayValues[highway] {
		return util.WayAccessCanSkip
	}

	if highway == "track" {
		trackType := way.GetTag("tracktype")
		if trackType != "" && !trackTypeValues[trackType] {
			return util.WayAccessCanSkip
		}
	}

	if p.blockFords && way.HasTag("ford", "yes") {
		return p.getAccessForFord(way)
	}

	return p.getAccessFromRestrictions(way)
}

func (p *AbstractAccessParser) getFerryAccess(way *reader.ReaderWay, highway string) util.WayAccess {
	firstValue := p.firstRestrictionValue(way)

	ferryAllowed := p.allowedValues[firstValue] ||
		(firstValue == "" && !way.HasTag("foot") && !way.HasTag("bicycle")) ||
		way.HasTag("hgv", "yes")

	if !ferryAllowed {
		return util.WayAccessCanSkip
	}
	if highway != "" {
		return util.WayAccessCanSkip
	}
	return util.WayAccessFerry
}

func (p *AbstractAccessParser) firstRestrictionValue(way *reader.ReaderWay) string {
	for _, key := range p.restrictionKeys {
		if val := way.GetTag(key); val != "" {
			return val
		}
	}
	return ""
}

func (p *AbstractAccessParser) getAccessForFord(way *reader.ReaderWay) util.WayAccess {
	for _, key := range p.restrictionKeys {
		if val := way.GetTag(key); val != "" && p.allowedValues[val] {
			return util.WayAccessWay
		}
	}
	return util.WayAccessCanSkip
}

func (p *AbstractAccessParser) getAccessFromRestrictions(way *reader.ReaderWay) util.WayAccess {
	for _, key := range p.restrictionKeys {
		val := way.GetTag(key)
		if val == "" {
			continue
		}
		if anySemicolonPartInSet(val, p.allowedValues) {
			return util.WayAccessWay
		}
		if anySemicolonPartInSet(val, p.restrictedValues) {
			if p.hasConditionalOverride(way, key) {
				return util.WayAccessWay
			}
			return util.WayAccessCanSkip
		}
	}

	if way.GetTag("service") == "emergency_access" {
		return util.WayAccessCanSkip
	}

	return util.WayAccessWay
}

// hasConditionalOverride checks whether a higher-priority conditional tag
// overrides a restriction found at restrictedKey.
func (p *AbstractAccessParser) hasConditionalOverride(way *reader.ReaderWay, restrictedKey string) bool {
	for _, key := range p.restrictionKeys {
		if key == restrictedKey {
			break
		}
		condVal := way.GetTag(key + ":conditional")
		if condVal == "" {
			continue
		}
		idx := strings.Index(condVal, "@")
		if idx < 0 {
			continue
		}
		if p.allowedValues[strings.TrimSpace(condVal[:idx])] {
			return true
		}
	}
	return false
}
