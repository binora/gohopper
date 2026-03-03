package parsers

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"gohopper/core/reader"
	"gohopper/core/routing/ev"
	"gohopper/core/storage"
)

const (
	MaxSpeedNone = -1.0
	// Minimum threshold in km/h below which we treat as invalid (3 mph ≈ 4.828).
	minMaxSpeed = 4.828
)

var maxSpeedPattern = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?)\s*(.*?)$`)

// OSMMaxSpeedParser parses the maxspeed, maxspeed:forward and maxspeed:backward
// tags and stores them in a DecimalEncodedValue.
type OSMMaxSpeedParser struct {
	maxSpeedEnc ev.DecimalEncodedValue
}

func NewOSMMaxSpeedParser(maxSpeedEnc ev.DecimalEncodedValue) *OSMMaxSpeedParser {
	return &OSMMaxSpeedParser{maxSpeedEnc: maxSpeedEnc}
}

func (p *OSMMaxSpeedParser) HandleWayTags(edgeID int, edgeIntAccess ev.EdgeIntAccess, way *reader.ReaderWay, _ *storage.IntsRef) {
	fwd := ParseMaxSpeed(way, false)
	bwd := ParseMaxSpeed(way, true)
	if fwd == ev.MaxSpeedMissing && bwd == ev.MaxSpeedMissing {
		return
	}
	if fwd != ev.MaxSpeedMissing {
		fwd = math.Min(fwd, ev.MaxSpeed150)
		p.maxSpeedEnc.SetDecimal(false, edgeID, edgeIntAccess, fwd)
	}
	if bwd != ev.MaxSpeedMissing {
		bwd = math.Min(bwd, ev.MaxSpeed150)
		p.maxSpeedEnc.SetDecimal(true, edgeID, edgeIntAccess, bwd)
	}
}

// ParseMaxSpeed returns the maxspeed for the given direction.
// Returns ev.MaxSpeedMissing if no valid maxspeed is found.
func ParseMaxSpeed(way *reader.ReaderWay, reverse bool) float64 {
	dirKey := "maxspeed:forward"
	if reverse {
		dirKey = "maxspeed:backward"
	}

	// Direction-specific tag takes priority.
	dirVal := way.GetTag(dirKey)
	if dirVal != "" {
		return ParseMaxspeedString(dirVal, way)
	}

	// For forward, fall back to maxspeed. For reverse only if no forward-specific tag exists.
	if reverse {
		fwdVal := way.GetTag("maxspeed:forward")
		if fwdVal != "" {
			// maxspeed:forward was set, so reverse uses maxspeed or missing.
			return ParseMaxspeedString(way.GetTag("maxspeed"), way)
		}
	}

	return ParseMaxspeedString(way.GetTag("maxspeed"), way)
}

// ParseMaxspeedString parses a maxspeed string value to km/h.
// Returns ev.MaxSpeedMissing for invalid values, MaxSpeedNone for "none".
func ParseMaxspeedString(str string, way *reader.ReaderWay) float64 {
	if str == "" {
		return ev.MaxSpeedMissing
	}

	str = strings.TrimSpace(str)
	if str == "none" || str == "unlimited" {
		return handleNone(way)
	}
	if str == "walk" || str == "living_street" {
		return 6
	}

	m := maxSpeedPattern.FindStringSubmatch(str)
	if m == nil {
		return ev.MaxSpeedMissing
	}

	val, err := strconv.ParseFloat(m[1], 64)
	if err != nil || val <= 0 {
		return ev.MaxSpeedMissing
	}

	unit := strings.TrimSpace(strings.ToLower(m[2]))
	switch {
	case unit == "" || unit == "km/h" || unit == "kmh" || unit == "kph":
		// km/h — no conversion
	case unit == "mph":
		val *= 1.609344
	case unit == "knots":
		val *= 1.852
	default:
		return ev.MaxSpeedMissing
	}

	if val < minMaxSpeed {
		return ev.MaxSpeedMissing
	}
	if val > ev.MaxSpeed150 {
		return ev.MaxSpeed150
	}
	return val
}

func handleNone(way *reader.ReaderWay) float64 {
	highway := ""
	if way != nil {
		highway = way.GetTag("highway")
	}
	switch highway {
	case "motorway", "motorway_link", "trunk", "trunk_link", "primary":
		return ev.MaxSpeed150
	default:
		return ev.MaxSpeedMissing
	}
}
