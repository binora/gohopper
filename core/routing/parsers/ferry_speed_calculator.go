package parsers

import (
	"fmt"
	"math"

	"gohopper/core/reader"
	"gohopper/core/routing/ev"
	"gohopper/core/storage"
)

// FerrySpeedCalculator computes ferry speed from duration tags and stores it.
type FerrySpeedCalculator struct {
	ferrySpeedEnc ev.DecimalEncodedValue
}

func NewFerrySpeedCalculator(ferrySpeedEnc ev.DecimalEncodedValue) *FerrySpeedCalculator {
	return &FerrySpeedCalculator{ferrySpeedEnc: ferrySpeedEnc}
}

// IsFerry returns true if the way represents a ferry or shuttle train route.
func IsFerry(way *reader.ReaderWay) bool {
	route := way.GetTag("route")
	switch route {
	case "ferry":
		return way.GetTag("ferry") != "no"
	case "shuttle_train":
		return way.GetTag("shuttle_train") != "no"
	default:
		return false
	}
}

func (c *FerrySpeedCalculator) HandleWayTags(edgeID int, edgeIntAccess ev.EdgeIntAccess, way *reader.ReaderWay, _ *storage.IntsRef) {
	speed := MinMax(GetFerrySpeed(way), c.ferrySpeedEnc)
	c.ferrySpeedEnc.SetDecimal(false, edgeID, edgeIntAccess, speed)
}

// GetFerrySpeed returns the ferry speed in km/h from way tags.
func GetFerrySpeed(way *reader.ReaderWay) float64 {
	if v := way.GetTagWithDefault("speed_from_duration", nil); v != nil {
		if speed, ok := toFloat64(v); ok {
			return math.Round(speed / 1.4)
		}
	}

	v := way.GetTagWithDefault("edge_distance", nil)
	if v == nil {
		panic(fmt.Sprintf("no speed_from_duration or edge_distance for ferry way: %d", way.GetID()))
	}
	dist, ok := toFloat64(v)
	if !ok {
		panic(fmt.Sprintf("invalid edge_distance for ferry way: %d", way.GetID()))
	}

	if dist < 500 {
		return 1
	}
	return 6
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

// MinMax clamps speed to [smallestNonZero, maxStorable].
func MinMax(speed float64, enc ev.DecimalEncodedValue) float64 {
	return math.Min(enc.GetMaxStorableDecimal(), math.Max(speed, enc.GetSmallestNonZeroValue()))
}
