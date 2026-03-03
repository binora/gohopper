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
	// MaxSpeedNone is the internal representation for maxspeed=none.
	MaxSpeedNone = -1.0
	// Minimum threshold in km/h below which we treat as invalid (3 mph ≈ 4.828).
	minMaxSpeed = 4.8
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
		p.maxSpeedEnc.SetDecimal(false, edgeID, edgeIntAccess, fwd)
	}
	if bwd != ev.MaxSpeedMissing {
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

	dirSpeed := parseMaxSpeedTag(way, dirKey)
	if dirSpeed != ev.MaxSpeedMissing {
		return dirSpeed
	}
	return parseMaxSpeedTag(way, "maxspeed")
}

// parseMaxSpeedTag parses a single maxspeed tag and handles the none+highway logic.
func parseMaxSpeedTag(way *reader.ReaderWay, tag string) float64 {
	maxSpeed := ParseMaxspeedString(way.GetTag(tag))
	if maxSpeed != ev.MaxSpeedMissing && maxSpeed != MaxSpeedNone {
		return math.Min(ev.MaxSpeed150, maxSpeed)
	}
	if maxSpeed == MaxSpeedNone && way.HasTag("highway", "motorway", "motorway_link", "trunk", "trunk_link", "primary") {
		return ev.MaxSpeed150
	}
	return ev.MaxSpeedMissing
}

// ParseMaxspeedString parses a maxspeed string value to km/h.
// Returns ev.MaxSpeedMissing for invalid values, MaxSpeedNone for "none"/"unlimited".
func ParseMaxspeedString(str string) float64 {
	if str == "" {
		return ev.MaxSpeedMissing
	}

	str = strings.TrimSpace(str)
	switch str {
	case "none", "unlimited":
		return MaxSpeedNone
	case "walk", "living_street":
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
	switch unit {
	case "", "km/h", "kmh", "kph":
		// already in km/h
	case "mph":
		val *= 1.609344
	case "knots":
		val *= 1.852
	default:
		return ev.MaxSpeedMissing
	}

	if val < minMaxSpeed {
		return ev.MaxSpeedMissing
	}
	return val
}
