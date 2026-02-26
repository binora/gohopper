package util

import "math"

const (
	// R is the mean radius of the earth in meters.
	R = 6371000.0
	// REQ is the radius of the earth at the equator in meters.
	REQ = 6378137.0
	// C is the circumference of the earth in meters.
	C = 2 * math.Pi * R
	// KmMile converts miles to kilometers.
	KmMile = 1.609344
	// MetersPerDegree is C / 360.
	MetersPerDegree = C / 360.0
)

// DistEarth is the singleton Haversine-based DistanceCalc.
var DistEarth DistanceCalc = &distanceCalcEarth{}

type distanceCalcEarth struct{}

func (d *distanceCalcEarth) CalcDist(fromLat, fromLon, toLat, toLon float64) float64 {
	nd := d.CalcNormalizedDistCoords(fromLat, fromLon, toLat, toLon)
	return R * 2 * math.Asin(math.Sqrt(nd))
}

func (d *distanceCalcEarth) CalcDist3D(fromLat, fromLon, fromEle, toLat, toLon, toEle float64) float64 {
	eleDelta := 0.0
	if hasElevationDiff(fromEle, toEle) {
		eleDelta = toEle - fromEle
	}
	length := d.CalcDist(fromLat, fromLon, toLat, toLon)
	return math.Hypot(eleDelta, length)
}

func (d *distanceCalcEarth) CalcDenormalizedDist(normedDist float64) float64 {
	return R * 2 * math.Asin(math.Sqrt(normedDist))
}

func (d *distanceCalcEarth) CalcNormalizedDist(dist float64) float64 {
	tmp := math.Sin(dist / 2 / R)
	return tmp * tmp
}

func (d *distanceCalcEarth) CalcNormalizedDistCoords(fromLat, fromLon, toLat, toLon float64) float64 {
	sinDeltaLat := math.Sin(toRad(toLat-fromLat) / 2)
	sinDeltaLon := math.Sin(toRad(toLon-fromLon) / 2)
	return sinDeltaLat*sinDeltaLat +
		sinDeltaLon*sinDeltaLon*math.Cos(toRad(fromLat))*math.Cos(toRad(toLat))
}

func (d *distanceCalcEarth) CalcCircumference(lat float64) float64 {
	return 2 * math.Pi * R * math.Cos(toRad(lat))
}

func (d *distanceCalcEarth) CreateBBox(lat, lon, radiusInMeter float64) BBox {
	if radiusInMeter <= 0 {
		panic("distance must not be zero or negative")
	}
	dLon := 360 / (d.CalcCircumference(lat) / radiusInMeter)
	dLat := 360 / (C / radiusInMeter)
	return NewBBox(lon-dLon, lon+dLon, lat-dLat, lat+dLat)
}

func (d *distanceCalcEarth) CalcNormalizedEdgeDistance(rLat, rLon, aLat, aLon, bLat, bLon float64) float64 {
	return calcNormEdgeDist(d, rLat, rLon, aLat, aLon, bLat, bLon)
}

func (d *distanceCalcEarth) CalcNormalizedEdgeDistance3D(rLat, rLon, rEle, aLat, aLon, aEle, bLat, bLon, bEle float64) float64 {
	return calcNormEdgeDist3D(d, rLat, rLon, rEle, aLat, aLon, aEle, bLat, bLon, bEle)
}

func (d *distanceCalcEarth) ValidEdgeDistance(rLat, rLon, aLat, aLon, bLat, bLon float64) bool {
	shrink := calcShrinkFactor(aLat, bLat)

	aLonN := aLon * shrink
	bLonN := bLon * shrink
	rLonN := rLon * shrink

	abX := bLonN - aLonN
	abY := bLat - aLat

	// dot(AR, AB) > 0: R is past A along AB
	arX := rLonN - aLonN
	arY := rLat - aLat
	dotARAB := arX*abX + arY*abY

	// dot(RB, AB) > 0: R is before B along AB
	rbX := bLonN - rLonN
	rbY := bLat - rLat
	dotRBAB := rbX*abX + rbY*abY

	return dotARAB > 0 && dotRBAB > 0
}

func (d *distanceCalcEarth) CalcCrossingPointToEdge(rLat, rLon, aLat, aLon, bLat, bLon float64) GHPoint {
	return calcCrossingPointToEdge(rLat, rLon, aLat, aLon, bLat, bLon)
}

func (d *distanceCalcEarth) ProjectCoordinate(lat, lon, distanceInMeter, headingClockwiseFromNorth float64) GHPoint {
	angularDist := distanceInMeter / R
	latRad := toRad(lat)
	lonRad := toRad(lon)
	headingRad := toRad(headingClockwiseFromNorth)

	projLat := math.Asin(math.Sin(latRad)*math.Cos(angularDist) +
		math.Cos(latRad)*math.Sin(angularDist)*math.Cos(headingRad))
	projLon := lonRad + math.Atan2(
		math.Sin(headingRad)*math.Sin(angularDist)*math.Cos(latRad),
		math.Cos(angularDist)-math.Sin(latRad)*math.Sin(projLat))

	projLon = math.Mod(projLon+3*math.Pi, 2*math.Pi) - math.Pi

	return GHPoint{Lat: toDeg(projLat), Lon: toDeg(projLon)}
}

func (d *distanceCalcEarth) IntermediatePoint(f, lat1, lon1, lat2, lon2 float64) GHPoint {
	lat1Rad := toRad(lat1)
	lon1Rad := toRad(lon1)
	lat2Rad := toRad(lat2)
	lon2Rad := toRad(lon2)

	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad
	cosLat1 := math.Cos(lat1Rad)
	cosLat2 := math.Cos(lat2Rad)
	sinHalfDLat := math.Sin(dLat / 2)
	sinHalfDLon := math.Sin(dLon / 2)

	a := sinHalfDLat*sinHalfDLat + cosLat1*cosLat2*sinHalfDLon*sinHalfDLon
	angularDist := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	sinDist := math.Sin(angularDist)

	if angularDist == 0 {
		return GHPoint{Lat: lat1, Lon: lon1}
	}

	A := math.Sin((1-f)*angularDist) / sinDist
	B := math.Sin(f*angularDist) / sinDist

	x := A*cosLat1*math.Cos(lon1Rad) + B*cosLat2*math.Cos(lon2Rad)
	y := A*cosLat1*math.Sin(lon1Rad) + B*cosLat2*math.Sin(lon2Rad)
	z := A*math.Sin(lat1Rad) + B*math.Sin(lat2Rad)

	midLat := toDeg(math.Atan2(z, math.Hypot(x, y)))
	midLon := toDeg(math.Atan2(y, x))

	return GHPoint{Lat: midLat, Lon: midLon}
}

func (d *distanceCalcEarth) IsCrossBoundary(lon1, lon2 float64) bool {
	return math.Abs(lon1-lon2) > 300
}

func (d *distanceCalcEarth) CalcPointListDistance(pl *PointList) float64 {
	return calcPLDist(d, pl)
}

// --- Shared package-level functions for proper dispatch ---

func calcShrinkFactor(aLat, bLat float64) float64 {
	return math.Cos(toRad((aLat + bLat) / 2))
}

func calcNormEdgeDist(dc DistanceCalc, rLat, rLon, aLat, aLon, bLat, bLon float64) float64 {
	shrink := calcShrinkFactor(aLat, bLat)

	aLonN := aLon * shrink
	bLonN := bLon * shrink
	rLonN := rLon * shrink

	dLon := bLonN - aLonN
	dLat := bLat - aLat

	if dLat == 0 {
		return dc.CalcNormalizedDistCoords(aLat, rLon, rLat, rLon)
	}
	if dLon == 0 {
		return dc.CalcNormalizedDistCoords(rLat, aLon, rLat, rLon)
	}

	norm := dLon*dLon + dLat*dLat
	factor := ((rLonN-aLonN)*dLon + (rLat-aLat)*dLat) / norm

	cLon := aLonN + factor*dLon
	cLat := aLat + factor*dLat
	return dc.CalcNormalizedDistCoords(cLat, cLon/shrink, rLat, rLon)
}

func calcNormEdgeDist3D(dc DistanceCalc, rLat, rLon, rEle, aLat, aLon, aEle, bLat, bLon, bEle float64) float64 {
	if math.IsNaN(rEle) || math.IsNaN(aEle) || math.IsNaN(bEle) {
		return dc.CalcNormalizedEdgeDistance(rLat, rLon, aLat, aLon, bLat, bLon)
	}

	shrink := calcShrinkFactor(aLat, bLat)
	invMPD := 1.0 / MetersPerDegree

	aLonN := aLon * shrink
	bLonN := bLon * shrink
	rLonN := rLon * shrink
	aEleN := aEle * invMPD
	bEleN := bEle * invMPD
	rEleN := rEle * invMPD

	dLon := bLonN - aLonN
	dLat := bLat - aLat
	dEle := bEleN - aEleN

	norm := dLon*dLon + dLat*dLat + dEle*dEle
	factor := ((rLonN-aLonN)*dLon + (rLat-aLat)*dLat + (rEleN-aEleN)*dEle) / norm
	if math.IsNaN(factor) {
		factor = 0
	}

	cLon := aLonN + factor*dLon
	cLat := aLat + factor*dLat
	cEle := (aEleN + factor*dEle) * MetersPerDegree
	return dc.CalcNormalizedDistCoords(cLat, cLon/shrink, rLat, rLon) + dc.CalcNormalizedDist(rEle-cEle)
}

func calcCrossingPointToEdge(rLat, rLon, aLat, aLon, bLat, bLon float64) GHPoint {
	shrink := calcShrinkFactor(aLat, bLat)

	aLonN := aLon * shrink
	bLonN := bLon * shrink
	rLonN := rLon * shrink

	dLon := bLonN - aLonN
	dLat := bLat - aLat

	if dLat == 0 {
		return GHPoint{Lat: aLat, Lon: rLon}
	}
	if dLon == 0 {
		return GHPoint{Lat: rLat, Lon: aLon}
	}

	norm := dLon*dLon + dLat*dLat
	factor := ((rLonN-aLonN)*dLon + (rLat-aLat)*dLat) / norm

	cLon := aLonN + factor*dLon
	cLat := aLat + factor*dLat
	return GHPoint{Lat: cLat, Lon: cLon / shrink}
}

func calcPLDist(dc DistanceCalc, pl *PointList) float64 {
	n := pl.Size()
	if n < 2 {
		return 0
	}
	is3D := pl.Is3D()
	var dist float64
	for i := 1; i < n; i++ {
		if is3D {
			dist += dc.CalcDist3D(
				pl.GetLat(i-1), pl.GetLon(i-1), pl.GetEle(i-1),
				pl.GetLat(i), pl.GetLon(i), pl.GetEle(i))
		} else {
			dist += dc.CalcDist(
				pl.GetLat(i-1), pl.GetLon(i-1),
				pl.GetLat(i), pl.GetLon(i))
		}
	}
	return dist
}

func hasElevationDiff(a, b float64) bool {
	return a != b && !math.IsNaN(a) && !math.IsNaN(b)
}

func toRad(deg float64) float64 { return deg * math.Pi / 180 }
func toDeg(rad float64) float64 { return rad * 180 / math.Pi }
