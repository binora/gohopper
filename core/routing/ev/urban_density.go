package ev

import "fmt"

var _ fmt.Stringer = UrbanDensity(0)

type UrbanDensity int

const (
	UrbanDensityRural UrbanDensity = iota
	UrbanDensityResidential
	UrbanDensityCity
	urbanDensityCount
)

const UrbanDensityKey = "urban_density"

var urbanDensityNames = [...]string{"rural", "residential", "city"}

func (u UrbanDensity) String() string {
	if u >= 0 && int(u) < len(urbanDensityNames) {
		return urbanDensityNames[u]
	}
	return "rural"
}

func UrbanDensityCreate() *EnumEncodedValue[UrbanDensity] {
	return NewEnumEncodedValue[UrbanDensity](UrbanDensityKey, enumSequence[UrbanDensity](int(urbanDensityCount)))
}
