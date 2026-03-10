package util

import (
	"math"
	"testing"
)

func TestXY2D_BasicProperties(t *testing.T) {
	// For order=2 (4x4 grid), the curve visits 16 cells (0..15).
	order := 2
	size := int64(1) << order
	seen := make(map[int64]bool)
	for x := range size {
		for y := range size {
			d := XY2D(order, x, y)
			if d < 0 || d >= size*size {
				t.Errorf("XY2D(%d, %d, %d) = %d, out of range [0, %d)", order, x, y, d, size*size)
			}
			if seen[d] {
				t.Errorf("XY2D(%d, %d, %d) = %d, duplicate distance", order, x, y, d)
			}
			seen[d] = true
		}
	}
	if len(seen) != int(size*size) {
		t.Errorf("expected %d unique distances, got %d", size*size, len(seen))
	}
}

func TestXY2D_KnownValues(t *testing.T) {
	// For order=2 (4x4), verify a few known Hilbert values.
	// (0,0) is always 0 for any order.
	if d := XY2D(2, 0, 0); d != 0 {
		t.Errorf("XY2D(2, 0, 0) = %d, want 0", d)
	}
}

func TestLatLonToHilbertIndex_Boundaries(t *testing.T) {
	order := 10
	maxD := int64(1)<<(2*order) - 1

	// Corner coordinates should produce valid indices.
	corners := [][2]float64{
		{-90, -180},
		{-90, 180},
		{90, -180},
		{90, 180},
		{0, 0},
	}
	for _, c := range corners {
		d := LatLonToHilbertIndex(c[0], c[1], order)
		if d < 0 || d > maxD {
			t.Errorf("LatLonToHilbertIndex(%f, %f, %d) = %d, out of range", c[0], c[1], order, d)
		}
	}
}

func TestLatLonToHilbertIndex_SpatialLocality(t *testing.T) {
	// Points close together should have closer Hilbert indices than distant points.
	order := 31
	// Two close points in Andorra
	d1 := LatLonToHilbertIndex(42.5063, 1.5218, order)
	d2 := LatLonToHilbertIndex(42.5073, 1.5228, order)
	// A distant point in Australia
	d3 := LatLonToHilbertIndex(-33.8688, 151.2093, order)

	closeDiff := math.Abs(float64(d1 - d2))
	farDiff := math.Abs(float64(d1 - d3))
	if closeDiff >= farDiff {
		t.Errorf("expected close points to have closer indices: close=%f, far=%f", closeDiff, farDiff)
	}
}

func TestLatLonToHilbertIndex_Order31(t *testing.T) {
	// Verify deterministic output for specific Andorra coordinates.
	d := LatLonToHilbertIndex(42.5063, 1.5218, 31)
	if d <= 0 {
		t.Errorf("expected positive Hilbert index for Andorra, got %d", d)
	}

	// Same input should always produce same output.
	d2 := LatLonToHilbertIndex(42.5063, 1.5218, 31)
	if d != d2 {
		t.Errorf("LatLonToHilbertIndex not deterministic: %d != %d", d, d2)
	}
}
