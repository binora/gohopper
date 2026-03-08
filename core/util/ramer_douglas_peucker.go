package util

import "math"

// RamerDouglasPeucker simplifies a list of 2D points which are not too far away.
// See http://en.wikipedia.org/wiki/Ramer–Douglas–Peucker_algorithm
type RamerDouglasPeucker struct {
	normedMaxDist        float64
	elevationMaxDistance float64
	maxDistance          float64
	calc                DistanceCalc
	approx              bool
}

func NewRamerDouglasPeucker() *RamerDouglasPeucker {
	rdp := &RamerDouglasPeucker{}
	rdp.SetApproximation(true)
	rdp.SetMaxDistance(1)
	rdp.SetElevationMaxDistance(math.MaxFloat64)
	return rdp
}

func (rdp *RamerDouglasPeucker) SetApproximation(a bool) {
	rdp.approx = a
	if a {
		rdp.calc = DistPlane
	} else {
		rdp.calc = DistEarth
	}
}

// SetMaxDistance sets maximum distance of discrepancy (from the normal way) in meters.
func (rdp *RamerDouglasPeucker) SetMaxDistance(dist float64) *RamerDouglasPeucker {
	rdp.normedMaxDist = rdp.calc.CalcNormalizedDist(dist)
	rdp.maxDistance = dist
	return rdp
}

// SetElevationMaxDistance sets maximum elevation distance of discrepancy in meters.
func (rdp *RamerDouglasPeucker) SetElevationMaxDistance(dist float64) *RamerDouglasPeucker {
	rdp.elevationMaxDistance = dist
	return rdp
}

// Simplify simplifies the entire PointList and returns the number of removed points.
func (rdp *RamerDouglasPeucker) Simplify(points *PointList) int {
	return rdp.SimplifyRange(points, 0, points.Size()-1, true)
}

// SimplifyFromTo simplifies a sub-range with compression.
func (rdp *RamerDouglasPeucker) SimplifyFromTo(points *PointList, fromIndex, lastIndex int) int {
	return rdp.SimplifyRange(points, fromIndex, lastIndex, true)
}

// SimplifyRange simplifies a part of the PointList. The fromIndex and lastIndex
// are guaranteed to be kept. If compress is true, NaN-marked points are removed.
func (rdp *RamerDouglasPeucker) SimplifyRange(points *PointList, fromIndex, lastIndex int, compress bool) int {
	removed := 0
	size := lastIndex - fromIndex
	if rdp.approx {
		delta := 500
		segments := size/delta + 1
		start := fromIndex
		for range segments {
			removed += rdp.subSimplify(points, start, min(lastIndex, start+delta))
			start += delta
		}
	} else {
		removed = rdp.subSimplify(points, fromIndex, lastIndex)
	}

	if removed > 0 && compress {
		RemoveNaN(points)
	}

	return removed
}

// subSimplify is the recursive core of the RDP algorithm.
// It keeps the points at fromIndex and lastIndex.
func (rdp *RamerDouglasPeucker) subSimplify(points *PointList, fromIndex, lastIndex int) int {
	if lastIndex-fromIndex < 2 {
		return 0
	}

	elevationFactor := rdp.maxDistance / rdp.elevationMaxDistance
	firstLat := points.GetLat(fromIndex)
	firstLon := points.GetLon(fromIndex)
	firstEle := points.GetEle(fromIndex)
	lastLat := points.GetLat(lastIndex)
	lastLon := points.GetLon(lastIndex)
	lastEle := points.GetEle(lastIndex)

	use3D := points.Is3D() && rdp.elevationMaxDistance < math.MaxFloat64 &&
		!math.IsNaN(firstEle) && !math.IsNaN(lastEle)

	indexWithMaxDist := -1
	maxDist := -1.0
	for i := fromIndex + 1; i < lastIndex; i++ {
		lat := points.GetLat(i)
		if math.IsNaN(lat) {
			continue
		}
		lon := points.GetLon(i)

		var dist float64
		if use3D {
			ele := points.GetEle(i)
			if !math.IsNaN(ele) {
				dist = rdp.calc.CalcNormalizedEdgeDistance3D(
					lat, lon, ele*elevationFactor,
					firstLat, firstLon, firstEle*elevationFactor,
					lastLat, lastLon, lastEle*elevationFactor)
			} else {
				dist = rdp.calc.CalcNormalizedEdgeDistance(lat, lon, firstLat, firstLon, lastLat, lastLon)
			}
		} else {
			dist = rdp.calc.CalcNormalizedEdgeDistance(lat, lon, firstLat, firstLon, lastLat, lastLon)
		}

		if maxDist < dist {
			indexWithMaxDist = i
			maxDist = dist
		}
	}

	if indexWithMaxDist < 0 {
		panic("maximum not found in sub-simplify range")
	}

	if maxDist < rdp.normedMaxDist {
		removed := lastIndex - fromIndex - 1
		for i := fromIndex + 1; i < lastIndex; i++ {
			points.Set(i, math.NaN(), math.NaN(), math.NaN())
		}
		return removed
	}
	return rdp.subSimplify(points, fromIndex, indexWithMaxDist) +
		rdp.subSimplify(points, indexWithMaxDist, lastIndex)
}

// RemoveNaN fills all entries of the point list that are NaN with the subsequent
// values (and therefore shortens the list).
func RemoveNaN(pl *PointList) {
	curr := 0
	for i := 0; i < pl.Size(); i++ {
		if !math.IsNaN(pl.GetLat(i)) {
			pl.Set(curr, pl.GetLat(i), pl.GetLon(i), pl.GetEle(i))
			curr++
		}
	}
	pl.TrimToSize(curr)
}
