package util

import "math"

// CalcOrientation returns the orientation of a line in radians relative to east.
// Result is in [-pi, +pi] where 0 is east.
func CalcOrientation(lat1, lon1, lat2, lon2 float64) float64 {
	shrinkFactor := math.Cos(toRad((lat1 + lat2) / 2))
	return math.Atan2(lat2-lat1, shrinkFactor*(lon2-lon1))
}

// CalcOrientationFast uses a fast atan2 approximation (Jim Shima).
func CalcOrientationFast(lat1, lon1, lat2, lon2 float64) float64 {
	shrinkFactor := math.Cos(toRad((lat1 + lat2) / 2))
	return fastAtan2(lat2-lat1, shrinkFactor*(lon2-lon1))
}

func fastAtan2(y, x float64) float64 {
	const (
		pi4  = math.Pi / 4
		pi34 = 3 * math.Pi / 4
	)
	absY := math.Abs(y) + 1e-10
	var r, angle float64
	if x < 0 {
		r = (x + absY) / (absY - x)
		angle = pi34
	} else {
		r = (x - absY) / (x + absY)
		angle = pi4
	}
	angle += (0.1963*r*r - 0.9817) * r
	if y < 0 {
		return -angle
	}
	return angle
}

// ConvertAzimuth2XAxisAngle converts north-based clockwise azimuth [0,360)
// into x-axis/east-based angle [-pi, pi].
func ConvertAzimuth2XAxisAngle(azimuth float64) float64 {
	if azimuth > 360 || azimuth < 0 {
		panic("azimuth must be in [0, 360]")
	}
	angleXY := math.Pi/2 - azimuth/180*math.Pi
	if angleXY < -math.Pi {
		angleXY += 2 * math.Pi
	}
	if angleXY > math.Pi {
		angleXY -= 2 * math.Pi
	}
	return angleXY
}

// AlignOrientation adjusts orientation so the difference to baseOrientation is <= pi.
func AlignOrientation(baseOrientation, orientation float64) float64 {
	if baseOrientation >= 0 {
		if orientation < -math.Pi+baseOrientation {
			return orientation + 2*math.Pi
		}
		return orientation
	}
	if orientation > math.Pi+baseOrientation {
		return orientation - 2*math.Pi
	}
	return orientation
}

// CalcAzimuth returns the azimuth in degrees (0=north, 90=east, 180=south, 270=west).
func CalcAzimuth(lat1, lon1, lat2, lon2 float64) float64 {
	orientation := math.Pi/2 - CalcOrientation(lat1, lon1, lat2, lon2)
	if orientation < 0 {
		orientation += 2 * math.Pi
	}
	return math.Mod(toDeg(Round4(orientation)), 360)
}

// Azimuth2CompassPoint converts azimuth degrees to a compass direction string.
func Azimuth2CompassPoint(azimuth float64) string {
	slice := 360.0 / 16
	switch {
	case azimuth < slice:
		return "N"
	case azimuth < slice*3:
		return "NE"
	case azimuth < slice*5:
		return "E"
	case azimuth < slice*7:
		return "SE"
	case azimuth < slice*9:
		return "S"
	case azimuth < slice*11:
		return "SW"
	case azimuth < slice*13:
		return "W"
	case azimuth < slice*15:
		return "NW"
	default:
		return "N"
	}
}

// IsClockwise returns true if vectors a→b→c follow clockwise order.
func IsClockwise(aX, aY, bX, bY, cX, cY float64) bool {
	angleDiff := (cX-aX)*(bY-aY) - (cY-aY)*(bX-aX)
	return angleDiff < 0
}
