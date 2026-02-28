package ev

import "fmt"

// Compile-time interface compliance check.
var _ fmt.Stringer = UrbanDensity(0)

// UrbanDensity defines the urban density classification.
type UrbanDensity int

const (
	UrbanDensityRural UrbanDensity = iota
	UrbanDensityResidential
	UrbanDensityCity
)

// UrbanDensityKey is the encoded value key for urban density.
const UrbanDensityKey = "urban_density"

// urbanDensityValues holds all UrbanDensity constants in ordinal order.
var urbanDensityValues = []UrbanDensity{
	UrbanDensityRural, UrbanDensityResidential, UrbanDensityCity,
}

// urbanDensityNames maps each UrbanDensity to its lowercase string representation.
var urbanDensityNames = [...]string{"rural", "residential", "city"}

// String returns the lowercase representation of the urban density.
func (u UrbanDensity) String() string {
	if u >= 0 && int(u) < len(urbanDensityNames) {
		return urbanDensityNames[u]
	}
	return "rural"
}

// UrbanDensityCreate creates an EnumEncodedValue for UrbanDensity.
func UrbanDensityCreate() *EnumEncodedValue[UrbanDensity] {
	return NewEnumEncodedValue[UrbanDensity](UrbanDensityKey, urbanDensityValues)
}
