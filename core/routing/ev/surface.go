package ev

import (
	"fmt"
	"strings"
)

// Compile-time interface compliance check.
var _ fmt.Stringer = Surface(0)

// Surface defines the road surface of an edge. If not tagged the value will
// be SurfaceMissing. All unknown surface tags get SurfaceOther.
type Surface int

const (
	SurfaceMissing Surface = iota
	SurfacePaved
	SurfaceAsphalt
	SurfaceConcrete
	SurfacePavingStones
	SurfaceCobblestone
	SurfaceUnpaved
	SurfaceCompacted
	SurfaceFineGravel
	SurfaceGravel
	SurfaceGround
	SurfaceDirt
	SurfaceGrass
	SurfaceSand
	SurfaceWood
	SurfaceOther
)

// SurfaceKey is the encoded value key for surface.
const SurfaceKey = "surface"

// surfaceValues holds all Surface constants in ordinal order.
var surfaceValues = []Surface{
	SurfaceMissing, SurfacePaved, SurfaceAsphalt, SurfaceConcrete,
	SurfacePavingStones, SurfaceCobblestone, SurfaceUnpaved, SurfaceCompacted,
	SurfaceFineGravel, SurfaceGravel, SurfaceGround, SurfaceDirt,
	SurfaceGrass, SurfaceSand, SurfaceWood, SurfaceOther,
}

// surfaceNames maps each Surface to its lowercase string representation.
var surfaceNames = [...]string{
	"missing", "paved", "asphalt", "concrete",
	"paving_stones", "cobblestone", "unpaved", "compacted",
	"fine_gravel", "gravel", "ground", "dirt",
	"grass", "sand", "wood", "other",
}

// surfaceMap maps surface name variants to Surface values, including aliases.
var surfaceMap map[string]Surface

func init() {
	surfaceMap = make(map[string]Surface, len(surfaceNames)+6)
	for i, name := range surfaceNames {
		s := Surface(i)
		if s == SurfaceMissing || s == SurfaceOther {
			continue
		}
		surfaceMap[name] = s
	}
	surfaceMap["metal"] = SurfacePaved
	surfaceMap["sett"] = SurfaceCobblestone
	surfaceMap["unhewn_cobblestone"] = SurfaceCobblestone
	surfaceMap["earth"] = SurfaceDirt
	surfaceMap["pebblestone"] = SurfaceGravel
	surfaceMap["grass_paver"] = SurfaceGrass
}

// String returns the lowercase representation of the surface.
func (s Surface) String() string {
	if s >= 0 && int(s) < len(surfaceNames) {
		return surfaceNames[s]
	}
	return "missing"
}

// SurfaceFind returns the Surface matching the given name (stripping any
// colon-separated suffix), or SurfaceOther for unrecognized tags.
// Returns SurfaceMissing for an empty name.
func SurfaceFind(name string) Surface {
	if name == "" {
		return SurfaceMissing
	}
	if idx := strings.Index(name, ":"); idx != -1 {
		name = name[:idx]
	}
	if v, ok := surfaceMap[name]; ok {
		return v
	}
	return SurfaceOther
}

// SurfaceCreate creates an EnumEncodedValue for Surface.
func SurfaceCreate() *EnumEncodedValue[Surface] {
	return NewEnumEncodedValue[Surface](SurfaceKey, surfaceValues)
}
