package util

// LatLonToHilbertIndex converts a lat/lon coordinate to a Hilbert curve index.
// The order parameter controls the grid resolution (2^order x 2^order).
func LatLonToHilbertIndex(lat, lon float64, order int) int64 {
	size := int64(1) << order
	x := clamp(int64((lon+180)/360*float64(size)), 0, size-1)
	y := clamp(int64((90-lat)/180*float64(size)), 0, size-1)
	return XY2D(order, x, y)
}

func clamp(v, lo, hi int64) int64 {
	return max(lo, min(hi, v))
}

// XY2D converts 2D grid coordinates to a Hilbert curve distance.
// order controls the recursion depth (grid is 2^order x 2^order).
func XY2D(order int, x, y int64) int64 {
	var d int64
	for s := int64(1) << (order - 1); s > 0; s >>= 1 {
		var rx, ry int
		if (x & s) > 0 {
			rx = 1
		}
		if (y & s) > 0 {
			ry = 1
		}
		d += s * s * int64((3*rx)^ry)
		if ry == 0 {
			if rx == 1 {
				x = s - 1 - x
				y = s - 1 - y
			}
			x, y = y, x
		}
	}
	return d
}
