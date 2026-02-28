package ev

import (
	"fmt"
	"strings"
)

var _ fmt.Stringer = Surface(0)

// Surface defines the road surface of an edge.
// SurfaceMissing for untagged, SurfaceOther for unrecognized tags.
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
	surfaceCount
)

const SurfaceKey = "surface"

var surfaceNames = [...]string{
	"missing", "paved", "asphalt", "concrete",
	"paving_stones", "cobblestone", "unpaved", "compacted",
	"fine_gravel", "gravel", "ground", "dirt",
	"grass", "sand", "wood", "other",
}

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

func (s Surface) String() string {
	if s >= 0 && int(s) < len(surfaceNames) {
		return surfaceNames[s]
	}
	return "missing"
}

// SurfaceFind returns the Surface matching the given name, stripping any
// colon-separated suffix. Returns SurfaceOther for unrecognized tags.
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

func SurfaceCreate() *EnumEncodedValue[Surface] {
	return NewEnumEncodedValue(SurfaceKey, enumSequence[Surface](int(surfaceCount)))
}
