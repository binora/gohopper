package util

import "math"

// DistPlane is the singleton plane-projection DistanceCalc, good for short distances.
var DistPlane DistanceCalc = &distancePlaneProjection{}

type distancePlaneProjection struct {
	distanceCalcEarth
}

func (d *distancePlaneProjection) CalcDist(fromLat, fromLon, toLat, toLon float64) float64 {
	nd := d.CalcNormalizedDistCoords(fromLat, fromLon, toLat, toLon)
	return R * math.Sqrt(nd)
}

func (d *distancePlaneProjection) CalcDist3D(fromLat, fromLon, fromEle, toLat, toLon, toEle float64) float64 {
	dEleNorm := 0.0
	if hasElevationDiff(fromEle, toEle) {
		dEleNorm = d.CalcNormalizedDist(toEle - fromEle)
	}
	nd := d.CalcNormalizedDistCoords(fromLat, fromLon, toLat, toLon)
	return R * math.Sqrt(nd+dEleNorm)
}

func (d *distancePlaneProjection) CalcDenormalizedDist(normedDist float64) float64 {
	return R * math.Sqrt(normedDist)
}

func (d *distancePlaneProjection) CalcNormalizedDist(dist float64) float64 {
	tmp := dist / R
	return tmp * tmp
}

func (d *distancePlaneProjection) CalcNormalizedDistCoords(fromLat, fromLon, toLat, toLon float64) float64 {
	dLat := toRad(toLat - fromLat)
	dLon := toRad(toLon - fromLon)
	left := math.Cos(toRad((fromLat+toLat)/2)) * dLon
	return dLat*dLat + left*left
}

// Methods below delegate to shared package-level functions for proper
// virtual dispatch — Go embedding doesn't provide it automatically.

func (d *distancePlaneProjection) CalcNormalizedEdgeDistance(rLat, rLon, aLat, aLon, bLat, bLon float64) float64 {
	return calcNormEdgeDist(d, rLat, rLon, aLat, aLon, bLat, bLon)
}

func (d *distancePlaneProjection) CalcNormalizedEdgeDistance3D(rLat, rLon, rEle, aLat, aLon, aEle, bLat, bLon, bEle float64) float64 {
	return calcNormEdgeDist3D(d, rLat, rLon, rEle, aLat, aLon, aEle, bLat, bLon, bEle)
}

func (d *distancePlaneProjection) CalcPointListDistance(pl *PointList) float64 {
	return calcPLDist(d, pl)
}
