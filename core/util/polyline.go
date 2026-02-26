package util

import "math"

// EncodePolyline encodes a PointList into a Google-compatible encoded polyline string.
func EncodePolyline(pl *PointList, includeElevation bool, multiplier float64) string {
	if multiplier < 1 {
		panic("multiplier cannot be smaller than 1")
	}
	out := make([]byte, 0, max(20, pl.Size()*3))
	prevLat, prevLon, prevEle := 0, 0, 0
	for i := 0; i < pl.Size(); i++ {
		num := int(math.Floor(pl.GetLat(i)*multiplier + 0.5))
		out = appendEncoded(out, int64(num-prevLat))
		prevLat = num

		num = int(math.Floor(pl.GetLon(i)*multiplier + 0.5))
		out = appendEncoded(out, int64(num-prevLon))
		prevLon = num

		if includeElevation {
			num = int(math.Floor(pl.GetEle(i)*100 + 0.5))
			out = appendEncoded(out, int64(num-prevEle))
			prevEle = num
		}
	}
	return string(out)
}

// EncodePolylineFromPoints encodes a slice of GHPoint (2D) into an encoded polyline string.
func EncodePolylineFromPoints(points []GHPoint, multiplier float64) string {
	if multiplier <= 0 {
		multiplier = 1e5
	}
	pl := NewPointListFromGHPoints(points)
	return EncodePolyline(pl, false, multiplier)
}

// DecodePolyline decodes a Google-compatible encoded polyline string into a PointList.
func DecodePolyline(encoded string, is3D bool, multiplier float64) *PointList {
	if multiplier < 1 {
		panic("multiplier cannot be smaller than 1")
	}
	initCap := max(10, len(encoded)/4)
	pl := NewPointList(initCap, is3D)

	idx := 0
	length := len(encoded)
	lat, lng, ele := 0, 0, 0

	for idx < length {
		// latitude
		shift, result := 0, 0
		for {
			b := int(encoded[idx]) - 63
			idx++
			result |= (b & 0x1f) << shift
			shift += 5
			if b < 0x20 {
				break
			}
		}
		if result&1 != 0 {
			lat += ^(result >> 1)
		} else {
			lat += result >> 1
		}

		// longitude
		shift, result = 0, 0
		for {
			b := int(encoded[idx]) - 63
			idx++
			result |= (b & 0x1f) << shift
			shift += 5
			if b < 0x20 {
				break
			}
		}
		if result&1 != 0 {
			lng += ^(result >> 1)
		} else {
			lng += result >> 1
		}

		if is3D {
			// elevation
			shift, result = 0, 0
			for {
				b := int(encoded[idx]) - 63
				idx++
				result |= (b & 0x1f) << shift
				shift += 5
				if b < 0x20 {
					break
				}
			}
			if result&1 != 0 {
				ele += ^(result >> 1)
			} else {
				ele += result >> 1
			}
			pl.Add3D(float64(lat)/multiplier, float64(lng)/multiplier, float64(ele)/100)
		} else {
			pl.Add(float64(lat)/multiplier, float64(lng)/multiplier)
		}
	}
	return pl
}

func appendEncoded(dst []byte, value int64) []byte {
	v := value << 1
	if value < 0 {
		v = ^v
	}
	for v >= 0x20 {
		dst = append(dst, byte((0x20|(v&0x1f))+63))
		v >>= 5
	}
	dst = append(dst, byte(v+63))
	return dst
}
