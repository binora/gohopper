package util

import "testing"

func TestEncodePolyline(t *testing.T) {
	points := []GHPoint{{Lat: 38.5, Lon: -120.2}, {Lat: 40.7, Lon: -120.95}, {Lat: 43.252, Lon: -126.453}}
	got := EncodePolyline(points, 1e5)
	want := "_p~iF~ps|U_ulLnnqC_mqNvxq`@"
	if got != want {
		t.Fatalf("unexpected polyline: got=%q want=%q", got, want)
	}
}
