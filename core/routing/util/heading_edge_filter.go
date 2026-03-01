package util

import (
	"math"

	ghutil "gohopper/core/util"
)

// HeadingEdgeFilter accepts edges whose azimuth is within tolerance of a
// given heading and that are close to the query point.
type HeadingEdgeFilter struct {
	heading            float64
	directedEdgeFilter DirectedEdgeFilter
	pointNearHeading   ghutil.GHPoint
}

func NewHeadingEdgeFilter(directedEdgeFilter DirectedEdgeFilter, heading float64, pointNearHeading ghutil.GHPoint) EdgeFilter {
	f := &HeadingEdgeFilter{
		heading:            heading,
		directedEdgeFilter: directedEdgeFilter,
		pointNearHeading:   pointNearHeading,
	}
	return f.Accept
}

func (f *HeadingEdgeFilter) Accept(edgeState ghutil.EdgeIteratorState) bool {
	const (
		tolerance   = 30.0
		maxDistance = 20.0
	)
	headingOfEdge := GetHeadingOfGeometryNearPoint(edgeState, f.pointNearHeading, maxDistance)
	if math.IsNaN(headingOfEdge) {
		return false
	}
	return (math.Abs(headingOfEdge-f.heading) < tolerance && f.directedEdgeFilter(edgeState, false)) ||
		(math.Abs(math.Mod(headingOfEdge+180, 360)-f.heading) < tolerance && f.directedEdgeFilter(edgeState, true))
}

// GetHeadingOfGeometryNearPoint returns the forward-direction heading (degrees)
// of the edge segment closest to point, or NaN if all segments exceed maxDistance.
func GetHeadingOfGeometryNearPoint(edgeState ghutil.EdgeIteratorState, point ghutil.GHPoint, maxDistance float64) float64 {
	calcDist := ghutil.DistEarth
	closestDistance := math.Inf(1)
	points := edgeState.FetchWayGeometry(ghutil.FetchModeAll)
	closestPoint := -1
	for i := 1; i < points.Size(); i++ {
		fromLat := points.GetLat(i - 1)
		fromLon := points.GetLon(i - 1)
		toLat := points.GetLat(i)
		toLon := points.GetLon(i)

		var distance float64
		if calcDist.ValidEdgeDistance(point.Lat, point.Lon, fromLat, fromLon, toLat, toLon) {
			distance = calcDist.CalcDenormalizedDist(
				calcDist.CalcNormalizedEdgeDistance(point.Lat, point.Lon, fromLat, fromLon, toLat, toLon))
		} else {
			distance = calcDist.CalcDist(fromLat, fromLon, point.Lat, point.Lon)
		}
		if i == points.Size()-1 {
			distance = math.Min(distance, calcDist.CalcDist(toLat, toLon, point.Lat, point.Lon))
		}
		if distance > maxDistance {
			continue
		}
		if distance < closestDistance {
			closestDistance = distance
			closestPoint = i
		}
	}
	if closestPoint < 0 {
		return math.NaN()
	}

	fromLat := points.GetLat(closestPoint - 1)
	fromLon := points.GetLon(closestPoint - 1)
	toLat := points.GetLat(closestPoint)
	toLon := points.GetLon(closestPoint)
	return ghutil.CalcAzimuth(fromLat, fromLon, toLat, toLon)
}
