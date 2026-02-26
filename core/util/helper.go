package util

import "math"

const (
	degreeFactor = 10_000_000
	eleFactor    = 1000.0
	maxEleUint   = int((10_000 + 1000) * eleFactor)
)

// Round rounds value to the specified number of decimal places.
func Round(value float64, decimalPlaces int) float64 {
	factor := math.Pow(10, float64(decimalPlaces))
	return math.Floor(value*factor+0.5) / factor
}

func Round2(value float64) float64 { return Round(value, 2) }
func Round4(value float64) float64 { return Round(value, 4) }
func Round6(value float64) float64 { return Round(value, 6) }

// EqualsEps returns true if d1 and d2 are within 1e-6 of each other.
func EqualsEps(d1, d2 float64) bool {
	return math.Abs(d1-d2) < 1e-6
}

// EqualsEpsCustom returns true if d1 and d2 are within epsilon of each other.
func EqualsEpsCustom(d1, d2, epsilon float64) bool {
	return math.Abs(d1-d2) < epsilon
}

// DegreeToInt converts a lat/lon degree to a compressed integer.
func DegreeToInt(deg float64) int32 {
	if deg >= math.MaxFloat64 {
		return math.MaxInt32
	}
	if deg <= -math.MaxFloat64 {
		return -math.MaxInt32
	}
	return int32(math.Floor(deg*degreeFactor + 0.5))
}

// IntToDegree converts a compressed integer back to a degree.
func IntToDegree(stored int32) float64 {
	if stored == math.MaxInt32 {
		return math.MaxFloat64
	}
	if stored == -math.MaxInt32 {
		return -math.MaxFloat64
	}
	return float64(stored) / degreeFactor
}

// EleToUInt converts elevation in meters to a compressed unsigned integer.
func EleToUInt(ele float64) int {
	if math.IsNaN(ele) {
		panic("elevation cannot be NaN")
	}
	if ele < -1000 {
		return 0
	}
	v := int(math.Floor((ele+1000)*eleFactor + 0.5))
	if v >= maxEleUint {
		return maxEleUint
	}
	return v
}

// UIntToEle converts a compressed unsigned integer back to elevation in meters.
func UIntToEle(integEle int) float64 {
	if integEle >= maxEleUint {
		return math.MaxFloat64
	}
	return float64(integEle)/eleFactor - 1000
}
