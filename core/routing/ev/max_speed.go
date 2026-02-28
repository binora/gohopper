package ev

import "math"

const MaxSpeedKey = "max_speed"

const (
	MaxSpeed150     = 150.0
	MaxSpeedMissing = math.MaxFloat64
)

func MaxSpeedCreate() DecimalEncodedValue {
	return NewDecimalEncodedValueImplFull(MaxSpeedKey, 7, 0, 2, false, true, true)
}
