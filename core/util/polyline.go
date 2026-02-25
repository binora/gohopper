package util

import "math"

func EncodePolyline(points []GHPoint, multiplier float64) string {
	if multiplier <= 0 {
		multiplier = 1e5
	}
	out := make([]byte, 0, len(points)*8)
	lastLat := int64(0)
	lastLon := int64(0)
	for _, p := range points {
		lat := int64(math.Round(p.Lat * multiplier))
		lon := int64(math.Round(p.Lon * multiplier))
		out = appendEncoded(out, lat-lastLat)
		out = appendEncoded(out, lon-lastLon)
		lastLat = lat
		lastLon = lon
	}
	return string(out)
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
