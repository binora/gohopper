package ev

import "fmt"

// Compile-time interface compliance check.
var _ fmt.Stringer = HazmatWater(0)

// HazmatWater defines general restrictions for the transport of goods
// through water protection areas. If not tagged it will be HazmatWaterYes.
type HazmatWater int

const (
	HazmatWaterYes HazmatWater = iota
	HazmatWaterPermissive
	HazmatWaterNo
)

// HazmatWaterKey is the encoded value key for hazmat water.
const HazmatWaterKey = "hazmat_water"

// hazmatWaterValues holds all HazmatWater constants in ordinal order.
var hazmatWaterValues = []HazmatWater{
	HazmatWaterYes, HazmatWaterPermissive, HazmatWaterNo,
}

// hazmatWaterNames maps each HazmatWater to its lowercase string representation.
var hazmatWaterNames = [...]string{"yes", "permissive", "no"}

// String returns the lowercase representation of the hazmat water value.
func (h HazmatWater) String() string {
	if h >= 0 && int(h) < len(hazmatWaterNames) {
		return hazmatWaterNames[h]
	}
	return "yes"
}

// HazmatWaterCreate creates an EnumEncodedValue for HazmatWater.
func HazmatWaterCreate() *EnumEncodedValue[HazmatWater] {
	return NewEnumEncodedValue[HazmatWater](HazmatWaterKey, hazmatWaterValues)
}
