package util

import (
	"math"
	"testing"
)

func TestOrientationExact(t *testing.T) {
	assertNear(t, 90.0, toDeg(CalcOrientation(0, 0, 1, 0)), 0.01)
	assertNear(t, 45.0, toDeg(CalcOrientation(0, 0, 1, 1)), 0.01)
	assertNear(t, 0.0, toDeg(CalcOrientation(0, 0, 0, 1)), 0.01)
	assertNear(t, -45.0, toDeg(CalcOrientation(0, 0, -1, 1)), 0.01)
	assertNear(t, -135.0, toDeg(CalcOrientation(0, 0, -1, -1)), 0.01)

	// symmetric
	assertNear(t, 90-32.76, toDeg(CalcOrientation(49.942, 11.580, 49.944, 11.582)), 0.01)
	assertNear(t, -90-32.76, toDeg(CalcOrientation(49.944, 11.582, 49.942, 11.580)), 0.01)
}

func TestOrientationFast(t *testing.T) {
	assertNear(t, 90.0, toDeg(CalcOrientationFast(0, 0, 1, 0)), 0.01)
	assertNear(t, 45.0, toDeg(CalcOrientationFast(0, 0, 1, 1)), 0.01)
	assertNear(t, 0.0, toDeg(CalcOrientationFast(0, 0, 0, 1)), 0.01)
	assertNear(t, -45.0, toDeg(CalcOrientationFast(0, 0, -1, 1)), 0.01)
	assertNear(t, -135.0, toDeg(CalcOrientationFast(0, 0, -1, -1)), 0.01)

	// symmetric — fast atan2 has slightly different precision
	assertNear(t, 90-32.92, toDeg(CalcOrientationFast(49.942, 11.580, 49.944, 11.582)), 0.01)
	assertNear(t, -90-32.92, toDeg(CalcOrientationFast(49.944, 11.582, 49.942, 11.580)), 0.01)
}

func TestAlignOrientation(t *testing.T) {
	assertNear(t, 90.0, toDeg(AlignOrientation(toRad(90), toRad(90))), 0.001)
	assertNear(t, 225.0, toDeg(AlignOrientation(toRad(90), toRad(-135))), 0.001)
	assertNear(t, -45.0, toDeg(AlignOrientation(toRad(-135), toRad(-45))), 0.001)
	assertNear(t, -270.0, toDeg(AlignOrientation(toRad(-135), toRad(90))), 0.001)
}

func TestCombined(t *testing.T) {
	orientation := CalcOrientation(52.414918, 13.244221, 52.415333, 13.243595)
	assertNear(t, 132.7, toDeg(AlignOrientation(0, orientation)), 1)

	orientation = CalcOrientation(52.414918, 13.244221, 52.414573, 13.243627)
	assertNear(t, -136.38, toDeg(AlignOrientation(0, orientation)), 1)
}

func TestCalcAzimuth(t *testing.T) {
	assertNear(t, 45.0, CalcAzimuth(0, 0, 1, 1), 0.001)
	assertNear(t, 90.0, CalcAzimuth(0, 0, 0, 1), 0.001)
	assertNear(t, 180.0, CalcAzimuth(0, 0, -1, 0), 0.001)
	assertNear(t, 270.0, CalcAzimuth(0, 0, 0, -1), 0.001)
	assertNear(t, 0.0, CalcAzimuth(49.942, 11.580, 49.944, 11.580), 0.001)
}

func TestAzimuthCompassPoint(t *testing.T) {
	if got := Azimuth2CompassPoint(199); got != "S" {
		t.Fatalf("got %q, want S", got)
	}
}

func TestFastAtan2(t *testing.T) {
	assertNear(t, 45, fastAtan2(5, 5)*180/math.Pi, 1e-2)
	assertNear(t, -45, fastAtan2(-5, 5)*180/math.Pi, 1e-2)
	assertNear(t, 11.14, fastAtan2(1, 5)*180/math.Pi, 1)
	assertNear(t, 180, fastAtan2(0, -5)*180/math.Pi, 1e-2)
	assertNear(t, -90, fastAtan2(-5, 0)*180/math.Pi, 1e-2)

	// Java testAtan2: reference against stdlib math.Atan2
	assertNear(t, 90, math.Atan2(1, 0)*180/math.Pi, 1e-2)
	assertNear(t, 90, fastAtan2(1, 0)*180/math.Pi, 1e-2)
}

func TestConvertAzimuth2XAxisAngle(t *testing.T) {
	assertNear(t, math.Pi/2, ConvertAzimuth2XAxisAngle(0), 1e-6)
	assertNear(t, math.Pi/2, math.Abs(ConvertAzimuth2XAxisAngle(360)), 1e-6)
	assertNear(t, 0, ConvertAzimuth2XAxisAngle(90), 1e-6)
	assertNear(t, -math.Pi/2, ConvertAzimuth2XAxisAngle(180), 1e-6)
	assertNear(t, math.Pi, math.Abs(ConvertAzimuth2XAxisAngle(270)), 1e-6)
	assertNear(t, -3*math.Pi/4, ConvertAzimuth2XAxisAngle(225), 1e-6)
	assertNear(t, 3*math.Pi/4, ConvertAzimuth2XAxisAngle(315), 1e-6)
}

func TestAzimuthConsistency(t *testing.T) {
	azimuth := CalcAzimuth(0, 0, 1, 1)
	radianXY := CalcOrientation(0, 0, 1, 1)
	radian2 := ConvertAzimuth2XAxisAngle(azimuth)
	assertNear(t, radianXY, radian2, 1e-3)

	azimuth = CalcAzimuth(0, 4, 1, 3)
	radianXY = CalcOrientation(0, 4, 1, 3)
	radian2 = ConvertAzimuth2XAxisAngle(azimuth)
	assertNear(t, radianXY, radian2, 1e-3)
}

func TestIsClockwise(t *testing.T) {
	assertTrue(t, IsClockwise(0.1, 1, 0.2, 0.8, 0.6, 0.3))
	assertTrue(t, IsClockwise(0.2, 0.8, 0.6, 0.3, 0.1, 1))
	assertTrue(t, IsClockwise(0.6, 0.3, 0.1, 1, 0.2, 0.8))
	assertFalse(t, IsClockwise(0.6, 0.3, 0.2, 0.8, 0.1, 1))
	assertFalse(t, IsClockwise(0.1, 1, 0.6, 0.3, 0.2, 0.8))
	assertFalse(t, IsClockwise(0.2, 0.8, 0.1, 1, 0.6, 0.3))
}
