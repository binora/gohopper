package util

import (
	"math"
	"testing"
)

func TestEncodePolylineFromPointsCompat(t *testing.T) {
	points := []GHPoint{{Lat: 38.5, Lon: -120.2}, {Lat: 40.7, Lon: -120.95}, {Lat: 43.252, Lon: -126.453}}
	got := EncodePolylineFromPoints(points, 1e5)
	want := "_p~iF~ps|U_ulLnnqC_mqNvxq`@"
	if got != want {
		t.Fatalf("unexpected polyline: got=%q want=%q", got, want)
	}
}

func TestDecodePolyline2D(t *testing.T) {
	pl := DecodePolyline("_p~iF~ps|U", false, 1e5)
	if pl.Size() != 1 {
		t.Fatalf("expected 1 point, got %d", pl.Size())
	}
	assertNear(t, 38.5, pl.GetLat(0), 1e-5)
	assertNear(t, -120.2, pl.GetLon(0), 1e-5)

	pl = DecodePolyline("_p~iF~ps|U_ulLnnqC_mqNvxq`@", false, 1e5)
	if pl.Size() != 3 {
		t.Fatalf("expected 3 points, got %d", pl.Size())
	}
	assertNear(t, 38.5, pl.GetLat(0), 1e-5)
	assertNear(t, -120.2, pl.GetLon(0), 1e-5)
	assertNear(t, 40.7, pl.GetLat(1), 1e-5)
	assertNear(t, -120.95, pl.GetLon(1), 1e-5)
	assertNear(t, 43.252, pl.GetLat(2), 1e-5)
	assertNear(t, -126.453, pl.GetLon(2), 1e-5)
}

func TestPolylineRoundtrip2D(t *testing.T) {
	poly := CreatePointList(38.5, -120.2, 40.7, -120.95, 43.252, -126.453)
	encoded := EncodePolyline(poly, false, 1e5)
	if encoded != "_p~iF~ps|U_ulLnnqC_mqNvxq`@" {
		t.Fatalf("unexpected encoding: %q", encoded)
	}

	decoded := DecodePolyline(encoded, false, 1e5)
	assertTrue(t, poly.Equals(decoded))
}

func TestPolylineRoundtripAdditional(t *testing.T) {
	pl := CreatePointList(50.3139, 10.612793, 50.04303, 9.497681)
	encoded := EncodePolyline(pl, false, 1e5)
	decoded := DecodePolyline(encoded, false, 1e5)
	if decoded.Size() != pl.Size() {
		t.Fatalf("size mismatch: %d != %d", decoded.Size(), pl.Size())
	}
	for i := 0; i < pl.Size(); i++ {
		assertNear(t, pl.GetLat(i), decoded.GetLat(i), 1e-5)
		assertNear(t, pl.GetLon(i), decoded.GetLon(i), 1e-5)
	}
}

func TestPolyline3DDecode(t *testing.T) {
	pl := DecodePolyline("_p~iF~ps|Uo}@", true, 1e5)
	if pl.Size() != 1 {
		t.Fatalf("expected 1 point, got %d", pl.Size())
	}
	assertNear(t, 38.5, pl.GetLat(0), 1e-5)
	assertNear(t, -120.2, pl.GetLon(0), 1e-5)
	assertNear(t, 10, pl.GetEle(0), 1e-2)
}

func TestPolyline3DEncodeDecode(t *testing.T) {
	pl := CreatePointList3D(38.5, -120.2, 10, 40.7, -120.95, 1.1, 43.252, -126.453, 0)
	encoded := EncodePolyline(pl, true, 1e5)
	decoded := DecodePolyline(encoded, true, 1e5)
	assertTrue(t, pl.Equals(decoded))
}

func TestPolyline3DRoundtrip(t *testing.T) {
	poly1 := CreatePointList3D(38.5, -120.2, 10)
	if EncodePolyline(poly1, true, 1e5) != "_p~iF~ps|Uo}@" {
		t.Fatalf("unexpected 3D encoding for single point")
	}

	poly := CreatePointList3D(38.5, -120.2, 10, 40.7, -120.95, -5, 43.252, -126.453, 0)
	encoded := EncodePolyline(poly, true, 1e5)
	if encoded != "_p~iF~ps|Uo}@_ulLnnqCv|A_mqNvxq`@g^" {
		t.Fatalf("unexpected 3D encoding: %q", encoded)
	}
}

func TestPolylineHighPrecision(t *testing.T) {
	pl := CreatePointList(47.827608, 12.123476, 47.827712, 12.123469)
	encoded := EncodePolyline(pl, false, 1e6)
	if encoded != "ohdfzAgt}bVoEL" {
		t.Fatalf("unexpected 1e6 encoding: %q", encoded)
	}
}

func FuzzPolylineRoundtrip(f *testing.F) {
	f.Add(38.5, -120.2, 40.7, -120.95, 43.252, -126.453)
	f.Add(0.0, 0.0, 0.0, 0.0, 0.0, 0.0)
	f.Add(-90.0, -180.0, 90.0, 180.0, 0.0, 0.0)

	f.Fuzz(func(t *testing.T, lat1, lon1, lat2, lon2, lat3, lon3 float64) {
		clamp := func(v, lo, hi float64) float64 {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return 0
			}
			if v < lo {
				return lo
			}
			if v > hi {
				return hi
			}
			return v
		}
		lat1 = clamp(lat1, -90, 90)
		lon1 = clamp(lon1, -180, 180)
		lat2 = clamp(lat2, -90, 90)
		lon2 = clamp(lon2, -180, 180)
		lat3 = clamp(lat3, -90, 90)
		lon3 = clamp(lon3, -180, 180)

		pl := CreatePointList(lat1, lon1, lat2, lon2, lat3, lon3)
		encoded := EncodePolyline(pl, false, 1e5)
		decoded := DecodePolyline(encoded, false, 1e5)
		if decoded.Size() != pl.Size() {
			t.Fatalf("size mismatch: %d != %d", decoded.Size(), pl.Size())
		}
		for i := 0; i < pl.Size(); i++ {
			if !EqualsEpsCustom(pl.GetLat(i), decoded.GetLat(i), 1e-5) {
				t.Fatalf("lat[%d] mismatch: %v != %v", i, pl.GetLat(i), decoded.GetLat(i))
			}
			if !EqualsEpsCustom(pl.GetLon(i), decoded.GetLon(i), 1e-5) {
				t.Fatalf("lon[%d] mismatch: %v != %v", i, pl.GetLon(i), decoded.GetLon(i))
			}
		}
	})
}
