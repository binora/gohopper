package ev

import "fmt"

// Compile-time interface compliance check.
var _ fmt.Stringer = Hazmat(0)

// Hazmat defines general restrictions for the transport of hazardous materials.
// If not tagged it will be HazmatYes.
type Hazmat int

const (
	HazmatYes Hazmat = iota
	HazmatNo
)

// HazmatKey is the encoded value key for hazmat.
const HazmatKey = "hazmat"

// hazmatValues holds all Hazmat constants in ordinal order.
var hazmatValues = []Hazmat{HazmatYes, HazmatNo}

// hazmatNames maps each Hazmat to its lowercase string representation.
var hazmatNames = [...]string{"yes", "no"}

// String returns the lowercase representation of the hazmat value.
func (h Hazmat) String() string {
	if h >= 0 && int(h) < len(hazmatNames) {
		return hazmatNames[h]
	}
	return "yes"
}

// HazmatCreate creates an EnumEncodedValue for Hazmat.
func HazmatCreate() *EnumEncodedValue[Hazmat] {
	return NewEnumEncodedValue[Hazmat](HazmatKey, hazmatValues)
}
