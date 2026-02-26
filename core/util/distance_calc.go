package util

// DistanceCalc provides geographic distance and geometry calculations.
type DistanceCalc interface {
	CalcDist(fromLat, fromLon, toLat, toLon float64) float64
	CalcDist3D(fromLat, fromLon, fromEle, toLat, toLon, toEle float64) float64
	CalcNormalizedDist(dist float64) float64
	CalcNormalizedDistCoords(fromLat, fromLon, toLat, toLon float64) float64
	CalcDenormalizedDist(normedDist float64) float64
	CalcCircumference(lat float64) float64
	CreateBBox(lat, lon, radiusInMeter float64) BBox
	ValidEdgeDistance(rLat, rLon, aLat, aLon, bLat, bLon float64) bool
	CalcNormalizedEdgeDistance(rLat, rLon, aLat, aLon, bLat, bLon float64) float64
	CalcNormalizedEdgeDistance3D(rLat, rLon, rEle, aLat, aLon, aEle, bLat, bLon, bEle float64) float64
	CalcCrossingPointToEdge(rLat, rLon, aLat, aLon, bLat, bLon float64) GHPoint
	ProjectCoordinate(lat, lon, distanceInMeter, headingClockwiseFromNorth float64) GHPoint
	IntermediatePoint(f, lat1, lon1, lat2, lon2 float64) GHPoint
	IsCrossBoundary(lon1, lon2 float64) bool
	CalcPointListDistance(pl *PointList) float64
}
