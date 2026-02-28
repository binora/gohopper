package ev

import "fmt"

var _ fmt.Stringer = HazmatWater(0)

type HazmatWater int

const (
	HazmatWaterYes HazmatWater = iota
	HazmatWaterPermissive
	HazmatWaterNo
	hazmatWaterCount
)

const HazmatWaterKey = "hazmat_water"

var hazmatWaterNames = [...]string{"yes", "permissive", "no"}

func (h HazmatWater) String() string {
	if h >= 0 && int(h) < len(hazmatWaterNames) {
		return hazmatWaterNames[h]
	}
	return "yes"
}

func HazmatWaterCreate() *EnumEncodedValue[HazmatWater] {
	return NewEnumEncodedValue[HazmatWater](HazmatWaterKey, enumSequence[HazmatWater](int(hazmatWaterCount)))
}
