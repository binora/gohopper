package util

import (
	"math"
	"testing"
)

func TestCalcCircumference(t *testing.T) {
	dc := DistEarth
	assertNear(t, C, dc.CalcCircumference(0), 1e-7)
}

func TestDistance(t *testing.T) {
	dc := DistEarth
	approx := DistPlane
	lat := float64(float32(24.235))
	lon := float64(float32(47.234))

	tests := []struct {
		dlat, dlon, expectedDist, earthDelta, planeDelta float64
	}{
		{-0.1, 0.1, 15051, 1, 1},
		{0.1, -0.1, 15046, 1, 1},
		{-1, 1, 150748, 1, 10},
		{1, -1, 150211, 1, 10},
		{-10, 10, 1527919, 1, 10000},
		{10, -10, 1474016, 1, 10000},
	}
	for _, tt := range tests {
		got := dc.CalcDist(lat, lon, lat+tt.dlat, lon+tt.dlon)
		assertNear(t, tt.expectedDist, got, tt.earthDelta)

		nd := dc.CalcNormalizedDistCoords(lat, lon, lat+tt.dlat, lon+tt.dlon)
		ndFromDist := dc.CalcNormalizedDist(tt.expectedDist)
		assertNear(t, ndFromDist, nd, 1)

		gotPlane := approx.CalcDist(lat, lon, lat+tt.dlat, lon+tt.dlon)
		assertNear(t, tt.expectedDist, gotPlane, tt.planeDelta)
	}

	// lon only
	assertNear(t, 1013735.28, dc.CalcDist(lat, lon, lat, lon-10), 1)
	assertNear(t, 1013735.28, approx.CalcDist(lat, lon, lat, lon-10), 1000)

	// lat only
	assertNear(t, 1111949.3, dc.CalcDist(lat, lon, lat+10, lon), 1)
	assertNear(t, 1111949.3, approx.CalcDist(lat, lon, lat+10, lon), 1)
}

func TestEdgeDistance(t *testing.T) {
	dc := DistEarth
	dist := dc.CalcNormalizedEdgeDistance(49.94241, 11.544356,
		49.937964, 11.541824,
		49.942272, 11.555643)
	expectedDist := dc.CalcNormalizedDistCoords(49.94241, 11.544356, 49.9394, 11.54681)
	assertNear(t, expectedDist, dist, 1e-4)

	// identical lats
	dist = dc.CalcNormalizedEdgeDistance(49.936299, 11.543992,
		49.9357, 11.543047,
		49.9357, 11.549227)
	expectedDist = dc.CalcNormalizedDistCoords(49.936299, 11.543992, 49.9357, 11.543992)
	assertNear(t, expectedDist, dist, 1e-4)
}

func TestEdgeDistance3d(t *testing.T) {
	dc := DistEarth
	dist := dc.CalcNormalizedEdgeDistance3D(49.94241, 11.544356, 0,
		49.937964, 11.541824, 0,
		49.942272, 11.555643, 0)
	expectedDist := dc.CalcNormalizedDistCoords(49.94241, 11.544356, 49.9394, 11.54681)
	assertNear(t, expectedDist, dist, 1e-4)

	// identical lats
	dist = dc.CalcNormalizedEdgeDistance3D(49.936299, 11.543992, 0,
		49.9357, 11.543047, 0,
		49.9357, 11.549227, 0)
	expectedDist = dc.CalcNormalizedDistCoords(49.936299, 11.543992, 49.9357, 11.543992)
	assertNear(t, expectedDist, dist, 1e-4)
}

func TestEdgeDistance3dEarth(t *testing.T) {
	dc := DistEarth
	dist := dc.CalcNormalizedEdgeDistance3D(0, 0.5, 10,
		0, 0, 0,
		0, 1, 0)
	assertNear(t, 10, dc.CalcDenormalizedDist(dist), 1e-4)
}

func TestEdgeDistance3dEarthNaN(t *testing.T) {
	dc := DistEarth
	dist := dc.CalcNormalizedEdgeDistance3D(0, 0.5, math.NaN(),
		0, 0, 0,
		0, 1, 0)
	assertNear(t, 0, dc.CalcDenormalizedDist(dist), 1e-4)
}

func TestEdgeDistance3dPlane(t *testing.T) {
	dc := DistPlane
	dist := dc.CalcNormalizedEdgeDistance3D(0, 0.5, 10,
		0, 0, 0,
		0, 1, 0)
	assertNear(t, 10, dc.CalcDenormalizedDist(dist), 1e-4)
}

func TestEdgeDistanceStartEndSame(t *testing.T) {
	dc := DistPlane
	// just change elevation
	dist := dc.CalcNormalizedEdgeDistance3D(0, 0, 10, 0, 0, 0, 0, 0, 0)
	assertNear(t, 10, dc.CalcDenormalizedDist(dist), 1e-4)
	// just change lat
	dist = dc.CalcNormalizedEdgeDistance3D(1, 0, 0, 0, 0, 0, 0, 0, 0)
	assertNear(t, MetersPerDegree, dc.CalcDenormalizedDist(dist), 1e-4)
	// just change lon
	dist = dc.CalcNormalizedEdgeDistance3D(0, 1, 0, 0, 0, 0, 0, 0, 0)
	assertNear(t, MetersPerDegree, dc.CalcDenormalizedDist(dist), 1e-4)
}

func TestEdgeDistanceStartEndDifferentElevation(t *testing.T) {
	dc := DistPlane
	// just change elevation
	dist := dc.CalcNormalizedEdgeDistance3D(0, 0, 10, 0, 0, 0, 0, 0, 1)
	assertNear(t, 0, dc.CalcDenormalizedDist(dist), 1e-4)
	// just change lat
	dist = dc.CalcNormalizedEdgeDistance3D(1, 0, 0, 0, 0, 0, 0, 0, 1)
	assertNear(t, MetersPerDegree, dc.CalcDenormalizedDist(dist), 1e-4)
	// just change lon
	dist = dc.CalcNormalizedEdgeDistance3D(0, 1, 0, 0, 0, 0, 0, 0, 1)
	assertNear(t, MetersPerDegree, dc.CalcDenormalizedDist(dist), 1e-4)
}

func TestValidEdgeDistance(t *testing.T) {
	dc := DistEarth
	assertTrue(t, dc.ValidEdgeDistance(49.94241, 11.544356, 49.937964, 11.541824, 49.942272, 11.555643))
	assertTrue(t, dc.ValidEdgeDistance(49.936624, 11.547636, 49.937964, 11.541824, 49.942272, 11.555643))
	assertTrue(t, dc.ValidEdgeDistance(49.940712, 11.556069, 49.937964, 11.541824, 49.942272, 11.555643))

	// left bottom
	assertFalse(t, dc.ValidEdgeDistance(49.935119, 11.541649, 49.937964, 11.541824, 49.942272, 11.555643))
	// left top
	assertFalse(t, dc.ValidEdgeDistance(49.939317, 11.539675, 49.937964, 11.541824, 49.942272, 11.555643))
	// right top
	assertFalse(t, dc.ValidEdgeDistance(49.944482, 11.555446, 49.937964, 11.541824, 49.942272, 11.555643))
	// right bottom
	assertFalse(t, dc.ValidEdgeDistance(49.94085, 11.557356, 49.937964, 11.541824, 49.942272, 11.555643))
}

func TestPrecisionBug(t *testing.T) {
	dist := DistPlane
	queryLat, queryLon := 42.56819, 1.603231
	lat16, lon16 := 42.56674481705006, 1.6023790821964834
	lat17, lon17 := 42.56694505140808, 1.6020622462495173
	lat18, lon18 := 42.56715199128878, 1.601682266630581

	assertNear(t, 171.487, dist.CalcDist(queryLat, queryLon, lat18, lon18), 1e-3)
	assertNear(t, 168.298, dist.CalcDist(queryLat, queryLon, lat17, lon17), 1e-3)
	assertNear(t, 175.188, dist.CalcDist(queryLat, queryLon, lat16, lon16), 1e-3)

	assertNear(t, 167.385, dist.CalcDenormalizedDist(dist.CalcNormalizedEdgeDistance(queryLat, queryLon, lat16, lon16, lat17, lon17)), 1e-3)
	assertNear(t, 168.213, dist.CalcDenormalizedDist(dist.CalcNormalizedEdgeDistance(queryLat, queryLon, lat17, lon17, lat18, lon18)), 1e-3)

	cp := dist.CalcCrossingPointToEdge(queryLat, queryLon, lat16, lon16, lat17, lon17)
	assertNear(t, 42.567048, cp.Lat, 1e-4)
	assertNear(t, 1.6019, cp.Lon, 1e-4)
}

func TestPrecisionBug2(t *testing.T) {
	dist := DistPlane
	queryLat, queryLon := 55.818994, 37.595354
	tmpLat, tmpLon := 55.81777239183573, 37.59598350366913
	wayLat, wayLon := 55.818839128736535, 37.5942968784488

	assertNear(t, 68.25, dist.CalcDist(wayLat, wayLon, queryLat, queryLon), 0.1)
	assertNear(t, 60.88, dist.CalcDenormalizedDist(dist.CalcNormalizedEdgeDistance(queryLat, queryLon, tmpLat, tmpLon, wayLat, wayLon)), 0.1)

	cp := dist.CalcCrossingPointToEdge(queryLat, queryLon, tmpLat, tmpLon, wayLat, wayLon)
	assertNear(t, 55.81863, cp.Lat, 1e-4)
	assertNear(t, 37.594626, cp.Lon, 1e-4)
}

func TestDistance3dEarth(t *testing.T) {
	dc := DistEarth
	assertNear(t, 1, dc.CalcDist3D(0, 0, 0, 0, 0, 1), 1e-6)
}

func TestDistance3dEarthNaN(t *testing.T) {
	dc := DistEarth
	assertNear(t, 0, dc.CalcDist3D(0, 0, 0, 0, 0, math.NaN()), 1e-6)
	assertNear(t, 0, dc.CalcDist3D(0, 0, math.NaN(), 0, 0, 10), 1e-6)
	assertNear(t, 0, dc.CalcDist3D(0, 0, math.NaN(), 0, 0, math.NaN()), 1e-6)
}

func TestDistance3dPlane(t *testing.T) {
	dc := DistPlane
	assertNear(t, 1, dc.CalcDist3D(0, 0, 0, 0, 0, 1), 1e-6)
	assertNear(t, 10, dc.CalcDist3D(0, 0, 0, 0, 0, 10), 1e-6)
}

func TestDistance3dPlaneNaN(t *testing.T) {
	dc := DistPlane
	assertNear(t, 0, dc.CalcDist3D(0, 0, 0, 0, 0, math.NaN()), 1e-6)
	assertNear(t, 0, dc.CalcDist3D(0, 0, math.NaN(), 0, 0, 10), 1e-6)
	assertNear(t, 0, dc.CalcDist3D(0, 0, math.NaN(), 0, 0, math.NaN()), 1e-6)
}

func TestIntermediatePoint(t *testing.T) {
	dc := DistEarth
	p := dc.IntermediatePoint(0, 0, 0, 0, 0)
	assertNear(t, 0, p.Lat, 1e-5)
	assertNear(t, 0, p.Lon, 1e-5)

	p = dc.IntermediatePoint(0.5, 0, 0, 10, 0)
	assertNear(t, 5, p.Lat, 1e-5)
	assertNear(t, 0, p.Lon, 1e-5)

	p = dc.IntermediatePoint(0.5, 0, 0, 0, 10)
	assertNear(t, 0, p.Lat, 1e-5)
	assertNear(t, 5, p.Lon, 1e-5)

	// cross international date line going west
	p = dc.IntermediatePoint(0.5, 45, -179, 45, 177)
	assertNear(t, 45, p.Lat, 1)
	assertNear(t, 179, p.Lon, 1e-5)

	// cross international date line going east
	p = dc.IntermediatePoint(0.5, 45, 179, 45, -177)
	assertNear(t, 45, p.Lat, 1)
	assertNear(t, -179, p.Lon, 1e-5)

	// cross north pole
	p = dc.IntermediatePoint(0.25, 45, -90, 45, 90)
	assertNear(t, 67.5, p.Lat, 1e-1)
	assertNear(t, -90, p.Lon, 1e-5)

	p = dc.IntermediatePoint(0.75, 45, -90, 45, 90)
	assertNear(t, 67.5, p.Lat, 1e-1)
	assertNear(t, 90, p.Lon, 1e-5)
}

func TestIsCrossBoundary(t *testing.T) {
	dc := DistEarth
	assertTrue(t, dc.IsCrossBoundary(-170, 170))
	assertFalse(t, dc.IsCrossBoundary(-10, 10))
}

func TestCalcPointListDistance(t *testing.T) {
	dc := DistEarth
	pl := NewPointList(3, false)
	pl.Add(0, 0)
	pl.Add(0, 1)
	pl.Add(0, 2)
	dist := dc.CalcPointListDistance(pl)
	expected := dc.CalcDist(0, 0, 0, 1) + dc.CalcDist(0, 1, 0, 2)
	assertNear(t, expected, dist, 1e-6)
}
