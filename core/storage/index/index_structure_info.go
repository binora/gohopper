package index

import (
	"fmt"
	"math"

	"gohopper/core/geohash"
	"gohopper/core/util"
)

// IndexStructureInfo holds the configuration of the spatial index tree: the
// number of entries per level, the bit-shifts per level, the pixel grid for
// line traversal, and the spatial key algorithm for encoding/decoding.
type IndexStructureInfo struct {
	Entries  []int
	Shifts   []byte
	Grid     *PixelGridTraversal
	KeyAlgo  *geohash.SpatialKeyAlgo
	Bounds   util.BBox
	Parts    int
}

// CreateIndexStructureInfo builds an IndexStructureInfo for the given bounding
// box and minimum resolution (in meters). It chooses the tree depth
// automatically so that each leaf cell is roughly minResolutionInMeter wide.
func CreateIndexStructureInfo(bounds util.BBox, minResolutionInMeter int) *IndexStructureInfo {
	// An empty LocationIndex must still be saveable/loadable, so fall back to
	// a small default extent when the bounds are invalid.
	if !bounds.IsValid() {
		bounds = util.NewBBox(-10.0, 10.0, -10.0, 10.0)
	}

	lat := math.Min(math.Abs(bounds.MaxLat), math.Abs(bounds.MinLat))
	maxDistInMeter := math.Max(
		(bounds.MaxLat-bounds.MinLat)/360*util.C,
		(bounds.MaxLon-bounds.MinLon)/360*util.DistEarth.CalcCircumference(lat),
	)
	tmp := maxDistInMeter / float64(minResolutionInMeter)
	tmp = tmp * tmp

	var entries []int
	// The last level is always 4 to reduce costs if only a single entry.
	tmp /= 4
	for tmp > 1 {
		var tmpNo int
		if tmp >= 16 {
			tmpNo = 16
		} else if tmp >= 4 {
			tmpNo = 4
		} else {
			break
		}
		entries = append(entries, tmpNo)
		tmp /= float64(tmpNo)
	}
	entries = append(entries, 4)

	if len(entries) < 1 {
		panic("depth needs to be at least 1")
	}

	shifts := make([]byte, len(entries))
	lastEntry := entries[0]
	for i, e := range entries {
		if lastEntry < e {
			panic(fmt.Sprintf("entries should decrease or stay but was: %v", entries))
		}
		lastEntry = e
		shifts[i] = getShift(e)
	}

	var shiftSum int
	parts := int64(1)
	for i, s := range shifts {
		shiftSum += int(s)
		parts *= int64(entries[i])
	}
	if shiftSum > 64 {
		panic("sum of all shifts does not fit into a long variable")
	}
	partsInt := int(math.Round(math.Sqrt(float64(parts))))

	return &IndexStructureInfo{
		Entries: entries,
		Shifts:  shifts,
		Grid:    NewPixelGridTraversal(partsInt, bounds),
		KeyAlgo: geohash.NewSpatialKeyAlgo(shiftSum, bounds),
		Bounds:  bounds,
		Parts:   partsInt,
	}
}

// getShift returns the number of bits needed to represent entries (log2).
func getShift(entries int) byte {
	b := byte(math.Round(math.Log2(float64(entries))))
	if b <= 0 {
		panic(fmt.Sprintf("invalid shift: %d", b))
	}
	return b
}

// DeltaLat returns the latitude span of a single grid cell.
func (info *IndexStructureInfo) DeltaLat() float64 {
	return (info.Bounds.MaxLat - info.Bounds.MinLat) / float64(info.Parts)
}

// DeltaLon returns the longitude span of a single grid cell.
func (info *IndexStructureInfo) DeltaLon() float64 {
	return (info.Bounds.MaxLon - info.Bounds.MinLon) / float64(info.Parts)
}
