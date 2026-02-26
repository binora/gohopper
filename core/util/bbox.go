package util

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// BBox is a bounding box following the ISO 19115 standard:
// minLon, maxLon, minLat (south), maxLat.
type BBox struct {
	MinLon float64
	MaxLon float64
	MinLat float64
	MaxLat float64
	MinEle float64
	MaxEle float64
	Is3D   bool
}

func NewBBox(minLon, maxLon, minLat, maxLat float64) BBox {
	return BBox{
		MinLon: minLon, MaxLon: maxLon,
		MinLat: minLat, MaxLat: maxLat,
		MinEle: math.NaN(), MaxEle: math.NaN(),
	}
}

func NewBBox3D(minLon, maxLon, minLat, maxLat, minEle, maxEle float64) BBox {
	return BBox{
		MinLon: minLon, MaxLon: maxLon,
		MinLat: minLat, MaxLat: maxLat,
		MinEle: minEle, MaxEle: maxEle,
		Is3D:   true,
	}
}

// CreateInverse returns a BBox prefilled with extreme values for progressive expansion via Update.
func CreateInverse(is3D bool) BBox {
	if is3D {
		return BBox{
			MinLon: math.MaxFloat64, MaxLon: -math.MaxFloat64,
			MinLat: math.MaxFloat64, MaxLat: -math.MaxFloat64,
			MinEle: math.MaxFloat64, MaxEle: -math.MaxFloat64,
			Is3D:   true,
		}
	}
	return BBox{
		MinLon: math.MaxFloat64, MaxLon: -math.MaxFloat64,
		MinLat: math.MaxFloat64, MaxLat: -math.MaxFloat64,
		MinEle: math.NaN(), MaxEle: math.NaN(),
	}
}

func (b *BBox) Update(lat, lon float64) {
	if lat > b.MaxLat {
		b.MaxLat = lat
	}
	if lat < b.MinLat {
		b.MinLat = lat
	}
	if lon > b.MaxLon {
		b.MaxLon = lon
	}
	if lon < b.MinLon {
		b.MinLon = lon
	}
}

func (b *BBox) Update3D(lat, lon, ele float64) {
	if !b.Is3D {
		panic("cannot update elevation on a 2D BBox")
	}
	if ele > b.MaxEle {
		b.MaxEle = ele
	}
	if ele < b.MinEle {
		b.MinEle = ele
	}
	b.Update(lat, lon)
}

func (b BBox) Contains(lat, lon float64) bool {
	return lat <= b.MaxLat && lat >= b.MinLat && lon <= b.MaxLon && lon >= b.MinLon
}

func (b BBox) ContainsBBox(other BBox) bool {
	return b.MaxLat >= other.MaxLat && b.MinLat <= other.MinLat &&
		b.MaxLon >= other.MaxLon && b.MinLon <= other.MinLon
}

func (b BBox) Intersects(other BBox) bool {
	return b.MinLon < other.MaxLon && b.MinLat < other.MaxLat &&
		other.MinLon < b.MaxLon && other.MinLat < b.MaxLat
}

func (b BBox) IntersectsCoords(minLon, maxLon, minLat, maxLat float64) bool {
	return b.MinLon < maxLon && b.MinLat < maxLat &&
		minLon < b.MaxLon && minLat < b.MaxLat
}

// CalculateIntersection returns the overlapping region of two BBoxes.
// Returns zero BBox and false if they do not intersect.
func (b BBox) CalculateIntersection(other BBox) (BBox, bool) {
	if !b.Intersects(other) {
		return BBox{}, false
	}
	return NewBBox(
		math.Max(b.MinLon, other.MinLon),
		math.Min(b.MaxLon, other.MaxLon),
		math.Max(b.MinLat, other.MinLat),
		math.Min(b.MaxLat, other.MaxLat),
	), true
}

func (b BBox) IsValid() bool {
	if b.MinLon >= b.MaxLon {
		return false
	}
	if b.MinLat >= b.MaxLat {
		return false
	}
	if b.Is3D {
		if b.MinEle > b.MaxEle {
			return false
		}
		if b.MaxEle == -math.MaxFloat64 || b.MinEle == math.MaxFloat64 {
			return false
		}
	}
	return b.MaxLat != -math.MaxFloat64 && b.MinLat != math.MaxFloat64 &&
		b.MaxLon != -math.MaxFloat64 && b.MinLon != math.MaxFloat64
}

func (b BBox) Equals(other BBox) bool {
	return EqualsEps(b.MinLat, other.MinLat) && EqualsEps(b.MaxLat, other.MaxLat) &&
		EqualsEps(b.MinLon, other.MinLon) && EqualsEps(b.MaxLon, other.MaxLon)
}

func (b BBox) String() string {
	s := fmt.Sprintf("%v,%v,%v,%v", b.MinLon, b.MaxLon, b.MinLat, b.MaxLat)
	if b.Is3D {
		s += fmt.Sprintf(",%v,%v", b.MinEle, b.MaxEle)
	}
	return s
}

// ToArray returns [minLon, minLat, maxLon, maxLat] for JSON serialization.
func (b BBox) ToArray() [4]float64 {
	return [4]float64{b.MinLon, b.MinLat, b.MaxLon, b.MaxLat}
}

func (b BBox) ToGeoJSON() []float64 {
	if b.Is3D {
		return []float64{
			Round6(b.MinLon), Round6(b.MinLat), Round2(b.MinEle),
			Round6(b.MaxLon), Round6(b.MaxLat), Round2(b.MaxEle),
		}
	}
	return []float64{Round6(b.MinLon), Round6(b.MinLat), Round6(b.MaxLon), Round6(b.MaxLat)}
}

// ParseTwoPoints parses "lat1,lon1,lat2,lon2" into a BBox.
func ParseTwoPoints(s string) (BBox, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return BBox{}, fmt.Errorf("BBox should have 4 parts but was %s", s)
	}
	vals := make([]float64, 4)
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return BBox{}, fmt.Errorf("invalid number in BBox: %w", err)
		}
		vals[i] = v
	}
	return FromPoints(vals[0], vals[1], vals[2], vals[3]), nil
}

// ParseBBoxString parses "minLon,maxLon,minLat,maxLat" into a BBox.
func ParseBBoxString(s string) (BBox, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return BBox{}, fmt.Errorf("BBox should have 4 parts but was %s", s)
	}
	vals := make([]float64, 4)
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return BBox{}, fmt.Errorf("invalid number in BBox: %w", err)
		}
		vals[i] = v
	}
	return NewBBox(vals[0], vals[1], vals[2], vals[3]), nil
}

// FromPoints creates a BBox from two arbitrary points, normalizing min/max.
func FromPoints(lat1, lon1, lat2, lon2 float64) BBox {
	if lat1 > lat2 {
		lat1, lat2 = lat2, lat1
	}
	if lon1 > lon2 {
		lon1, lon2 = lon2, lon1
	}
	return NewBBox(lon1, lon2, lat1, lat2)
}

// CalcBBox computes the bounding box from a slice of GHPoints.
func CalcBBox(points []GHPoint) BBox {
	if len(points) == 0 {
		return BBox{}
	}
	b := BBox{
		MinLon: points[0].Lon, MaxLon: points[0].Lon,
		MinLat: points[0].Lat, MaxLat: points[0].Lat,
		MinEle: math.NaN(), MaxEle: math.NaN(),
	}
	for i := 1; i < len(points); i++ {
		b.Update(points[i].Lat, points[i].Lon)
	}
	return b
}
