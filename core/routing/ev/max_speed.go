package ev

import "math"

// MaxSpeedKey is the encoded value key for maximum speed.
const MaxSpeedKey = "max_speed"

// MaxSpeed150 is 150 km/h, a commonly used speed cap.
// MaxSpeedMissing indicates that no speed limit is tagged.
const (
	MaxSpeed150     = 150.0
	MaxSpeedMissing = math.MaxFloat64
)

// MaxSpeedCreate creates a DecimalEncodedValue for maximum speed (km/h).
func MaxSpeedCreate() DecimalEncodedValue {
	return NewDecimalEncodedValueImplFull(MaxSpeedKey, 7, 0, 2, false, true, true)
}
