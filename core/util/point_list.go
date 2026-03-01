package util

import (
	"fmt"
	"math"
	"slices"
	"strings"
)

// PointList is a slim list storing lat/lon/ele without object overhead per point.
type PointList struct {
	lats      []float64
	lons      []float64
	eles      []float64
	is3D      bool
	immutable bool
}

func NewPointList(cap int, is3D bool) *PointList {
	pl := &PointList{
		lats: make([]float64, 0, cap),
		lons: make([]float64, 0, cap),
		is3D: is3D,
	}
	if is3D {
		pl.eles = make([]float64, 0, cap)
	}
	return pl
}

// NewPointListFromGHPoints creates a 2D PointList from a slice of GHPoints.
func NewPointListFromGHPoints(points []GHPoint) *PointList {
	pl := NewPointList(len(points), false)
	for _, p := range points {
		pl.Add(p.Lat, p.Lon)
	}
	return pl
}

func (pl *PointList) ensureMutable() {
	if pl.immutable {
		panic("cannot change an immutable PointList")
	}
}

func (pl *PointList) Add(lat, lon float64) {
	pl.ensureMutable()
	if pl.is3D {
		panic("cannot add point without elevation data in 3D mode")
	}
	pl.lats = append(pl.lats, lat)
	pl.lons = append(pl.lons, lon)
}

func (pl *PointList) Add3D(lat, lon, ele float64) {
	pl.ensureMutable()
	pl.lats = append(pl.lats, lat)
	pl.lons = append(pl.lons, lon)
	if pl.is3D {
		pl.eles = append(pl.eles, ele)
	} else if !math.IsNaN(ele) {
		panic(fmt.Sprintf("this is a 2D list, cannot store elevation: %v", ele))
	}
}

func (pl *PointList) AddFrom(other *PointList, index int) {
	pl.ensureMutable()
	if pl.is3D {
		pl.Add3D(other.GetLat(index), other.GetLon(index), other.GetEle(index))
	} else {
		pl.Add(other.GetLat(index), other.GetLon(index))
	}
}

func (pl *PointList) AddPointList(other *PointList) {
	pl.ensureMutable()
	pl.lats = append(pl.lats, other.lats...)
	pl.lons = append(pl.lons, other.lons...)
	if pl.is3D {
		if other.is3D {
			pl.eles = append(pl.eles, other.eles...)
		} else {
			// Other is 2D; fill with NaN for each added point.
			for range other.lats {
				pl.eles = append(pl.eles, math.NaN())
			}
		}
	}
}

func (pl *PointList) Set(index int, lat, lon, ele float64) {
	pl.ensureMutable()
	pl.checkIndex(index)
	pl.lats[index] = lat
	pl.lons[index] = lon
	if pl.is3D {
		pl.eles[index] = ele
	} else if !math.IsNaN(ele) {
		panic(fmt.Sprintf("this is a 2D list, cannot store elevation: %v", ele))
	}
}

func (pl *PointList) checkIndex(index int) {
	if index >= len(pl.lats) {
		panic(fmt.Sprintf("index %d out of bounds for size %d", index, len(pl.lats)))
	}
}

func (pl *PointList) GetLat(index int) float64 {
	pl.checkIndex(index)
	return pl.lats[index]
}

func (pl *PointList) GetLon(index int) float64 {
	pl.checkIndex(index)
	return pl.lons[index]
}

func (pl *PointList) GetEle(index int) float64 {
	pl.checkIndex(index)
	if !pl.is3D {
		return math.NaN()
	}
	return pl.eles[index]
}

func (pl *PointList) Get(index int) GHPoint3D {
	return GHPoint3D{
		GHPoint: GHPoint{Lat: pl.GetLat(index), Lon: pl.GetLon(index)},
		Ele:     pl.GetEle(index),
	}
}

func (pl *PointList) Size() int        { return len(pl.lats) }
func (pl *PointList) IsEmpty() bool     { return len(pl.lats) == 0 }
func (pl *PointList) Is3D() bool        { return pl.is3D }
func (pl *PointList) IsImmutable() bool { return pl.immutable }

func (pl *PointList) MakeImmutable() *PointList {
	pl.immutable = true
	return pl
}

func (pl *PointList) Reverse() {
	pl.ensureMutable()
	slices.Reverse(pl.lats)
	slices.Reverse(pl.lons)
	if pl.is3D {
		slices.Reverse(pl.eles)
	}
}

func (pl *PointList) Clone(reverse bool) *PointList {
	c := pl.Copy(0, len(pl.lats))
	if reverse {
		c.Reverse()
	}
	return c
}

func (pl *PointList) Copy(from, end int) *PointList {
	if from > end {
		panic("from must be smaller or equal to end")
	}
	if from < 0 || end > len(pl.lats) {
		panic(fmt.Sprintf("illegal interval: %d, %d, size:%d", from, end, len(pl.lats)))
	}
	c := &PointList{
		lats: append([]float64(nil), pl.lats[from:end]...),
		lons: append([]float64(nil), pl.lons[from:end]...),
		is3D: pl.is3D,
	}
	if pl.is3D {
		c.eles = append([]float64(nil), pl.eles[from:end]...)
	}
	return c
}

func (pl *PointList) RemoveLastPoint() {
	pl.ensureMutable()
	n := len(pl.lats)
	if n == 0 {
		panic("cannot remove last point from empty PointList")
	}
	pl.lats = pl.lats[:n-1]
	pl.lons = pl.lons[:n-1]
	if pl.is3D {
		pl.eles = pl.eles[:n-1]
	}
}

func (pl *PointList) TrimToSize(newSize int) {
	pl.ensureMutable()
	if newSize > len(pl.lats) {
		panic("new size needs be smaller than old size")
	}
	pl.lats = pl.lats[:newSize]
	pl.lons = pl.lons[:newSize]
	if pl.is3D {
		pl.eles = pl.eles[:newSize]
	}
}

func (pl *PointList) Clear() {
	pl.ensureMutable()
	pl.lats = pl.lats[:0]
	pl.lons = pl.lons[:0]
	if pl.is3D {
		pl.eles = pl.eles[:0]
	}
}

func (pl *PointList) Equals(other *PointList) bool {
	if pl == nil && other == nil {
		return true
	}
	if pl == nil || other == nil {
		return false
	}
	if pl.IsEmpty() && other.IsEmpty() {
		return true
	}
	if len(pl.lats) != len(other.lats) || pl.is3D != other.is3D {
		return false
	}
	for i := range pl.lats {
		if !EqualsEps(pl.lats[i], other.lats[i]) || !EqualsEps(pl.lons[i], other.lons[i]) {
			return false
		}
		if pl.is3D && !EqualsEpsCustom(pl.eles[i], other.eles[i], 0.01) {
			return false
		}
	}
	return true
}

func (pl *PointList) String() string {
	var sb strings.Builder
	for i := range pl.lats {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteByte('(')
		fmt.Fprintf(&sb, "%v,%v", pl.lats[i], pl.lons[i])
		if pl.is3D {
			fmt.Fprintf(&sb, ",%v", pl.eles[i])
		}
		sb.WriteByte(')')
	}
	return sb.String()
}

// ToGHPoints converts to a slice of GHPoint (2D only).
func (pl *PointList) ToGHPoints() []GHPoint {
	pts := make([]GHPoint, len(pl.lats))
	for i := range pl.lats {
		pts[i] = GHPoint{Lat: pl.lats[i], Lon: pl.lons[i]}
	}
	return pts
}

// CreatePointList is a convenience to create a 2D PointList from lat,lon pairs.
func CreatePointList(latLons ...float64) *PointList {
	if len(latLons)%2 != 0 {
		panic("list should consist of lat,lon pairs")
	}
	n := len(latLons) / 2
	pl := NewPointList(n, false)
	for i := 0; i < n; i++ {
		pl.Add(latLons[2*i], latLons[2*i+1])
	}
	return pl
}

// CreatePointList3D is a convenience to create a 3D PointList from lat,lon,ele triples.
func CreatePointList3D(latLonEles ...float64) *PointList {
	if len(latLonEles)%3 != 0 {
		panic("list should consist of lat,lon,ele tuples")
	}
	n := len(latLonEles) / 3
	pl := NewPointList(n, true)
	for i := 0; i < n; i++ {
		pl.Add3D(latLonEles[3*i], latLonEles[3*i+1], latLonEles[3*i+2])
	}
	return pl
}
