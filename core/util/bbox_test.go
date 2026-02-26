package util

import (
	"math"
	"testing"
)

func TestBBoxCreate(t *testing.T) {
	b := DistEarth.CreateBBox(52, 10, 100000)

	assertNear(t, 52.8993, b.MaxLat, 1e-4)
	assertNear(t, 8.5393, b.MinLon, 1e-4)
	assertNear(t, 51.1007, b.MinLat, 1e-4)
	assertNear(t, 11.4607, b.MaxLon, 1e-4)
}

func TestBBoxContains(t *testing.T) {
	assertTrue(t, NewBBox(1, 2, 0, 1).ContainsBBox(NewBBox(1, 2, 0, 1)))
	assertTrue(t, NewBBox(1, 2, 0, 1).ContainsBBox(NewBBox(1.5, 2, 0.5, 1)))
	assertFalse(t, NewBBox(1, 2, 0, 0.5).ContainsBBox(NewBBox(1.5, 2, 0.5, 1)))
}

func TestBBoxIntersect(t *testing.T) {
	assertTrue(t, NewBBox(12, 15, 12, 15).Intersects(NewBBox(13, 14, 11, 16)))
	assertTrue(t, NewBBox(2, 6, 6, 11).Intersects(NewBBox(3, 5, 5, 12)))
	assertTrue(t, NewBBox(6, 11, 6, 11).Intersects(NewBBox(7, 10, 5, 12)))
}

func TestBBoxCalculateIntersection(t *testing.T) {
	b1 := NewBBox(0, 2, 0, 1)
	b2 := NewBBox(-1, 1, -1, 2)
	expected := NewBBox(0, 1, 0, 1)

	result, ok := b1.CalculateIntersection(b2)
	assertTrue(t, ok)
	assertTrue(t, expected.Equals(result))

	// No intersection
	b2 = NewBBox(100, 200, 100, 200)
	_, ok = b1.CalculateIntersection(b2)
	assertFalse(t, ok)

	// Real example
	b1 = NewBBox(8.8591, 9.9111, 48.3145, 48.8518)
	b2 = NewBBox(5.8524, 17.1483, 46.3786, 55.0653)
	result, ok = b1.CalculateIntersection(b2)
	assertTrue(t, ok)
	assertTrue(t, b1.Equals(result))
}

func TestParseTwoPoints(t *testing.T) {
	got, err := ParseTwoPoints("1,2,3,4")
	assertNoErr(t, err)
	expected := NewBBox(2, 4, 1, 3)
	assertTrue(t, expected.Equals(got))

	// stable: reversed lat order
	got, err = ParseTwoPoints("3,2,1,4")
	assertNoErr(t, err)
	assertTrue(t, expected.Equals(got))
}

func TestParseBBoxString(t *testing.T) {
	got, err := ParseBBoxString("2,4,1,3")
	assertNoErr(t, err)
	expected := NewBBox(2, 4, 1, 3)
	assertTrue(t, expected.Equals(got))
}

func TestBBoxIsValid(t *testing.T) {
	assertTrue(t, NewBBox(1, 2, 0, 1).IsValid())
	assertFalse(t, NewBBox(2, 1, 0, 1).IsValid())       // minLon > maxLon
	assertFalse(t, NewBBox(1, 2, 1, 0).IsValid())        // minLat > maxLat
	assertFalse(t, CreateInverse(false).IsValid())        // inverse is not valid until populated
}

func TestBBox3D(t *testing.T) {
	b := NewBBox3D(1, 2, 0, 1, 100, 200)
	assertTrue(t, b.Is3D)
	assertTrue(t, b.IsValid())

	b2 := NewBBox3D(1, 2, 0, 1, 200, 100)
	assertFalse(t, b2.IsValid()) // minEle > maxEle
}

func TestBBoxFromPoints(t *testing.T) {
	b := FromPoints(52.0, 13.0, 48.0, 10.0)
	assertNear(t, 48.0, b.MinLat, 1e-10)
	assertNear(t, 52.0, b.MaxLat, 1e-10)
	assertNear(t, 10.0, b.MinLon, 1e-10)
	assertNear(t, 13.0, b.MaxLon, 1e-10)
}

func TestCreateInverse(t *testing.T) {
	b := CreateInverse(false)
	if b.MinLon != math.MaxFloat64 {
		t.Fatal("MinLon should be MaxFloat64")
	}
	b.Update(52, 13)
	assertNear(t, 52, b.MinLat, 1e-10)
	assertNear(t, 52, b.MaxLat, 1e-10)
	assertNear(t, 13, b.MinLon, 1e-10)
	assertNear(t, 13, b.MaxLon, 1e-10)
}

// helpers

func assertNear(t *testing.T, expected, actual, delta float64) {
	t.Helper()
	if math.Abs(expected-actual) > delta {
		t.Fatalf("expected %v ± %v, got %v", expected, delta, actual)
	}
}

func assertTrue(t *testing.T, v bool) {
	t.Helper()
	if !v {
		t.Fatal("expected true")
	}
}

func assertFalse(t *testing.T, v bool) {
	t.Helper()
	if v {
		t.Fatal("expected false")
	}
}

func assertNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
