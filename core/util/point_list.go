package util

import (
	"fmt"
	"math"
	"strings"
)

// PointList is a slim list storing lat/lon/ele without object overhead per point.
type PointList struct {
	lats      []float64
	lons      []float64
	eles      []float64
	size      int
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
	pl.size++
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
	pl.size++
}

func (pl *PointList) AddPointList(other *PointList) {
	pl.ensureMutable()
	for i := 0; i < other.size; i++ {
		pl.lats = append(pl.lats, other.lats[i])
		pl.lons = append(pl.lons, other.lons[i])
		if pl.is3D {
			pl.eles = append(pl.eles, other.GetEle(i))
		}
	}
	pl.size += other.size
}

func (pl *PointList) Set(index int, lat, lon, ele float64) {
	pl.ensureMutable()
	if index >= pl.size {
		panic(fmt.Sprintf("index %d out of bounds for size %d", index, pl.size))
	}
	pl.lats[index] = lat
	pl.lons[index] = lon
	if pl.is3D {
		pl.eles[index] = ele
	} else if !math.IsNaN(ele) {
		panic(fmt.Sprintf("this is a 2D list, cannot store elevation: %v", ele))
	}
}

func (pl *PointList) GetLat(index int) float64 {
	if index >= pl.size {
		panic(fmt.Sprintf("index %d out of bounds for size %d", index, pl.size))
	}
	return pl.lats[index]
}

func (pl *PointList) GetLon(index int) float64 {
	if index >= pl.size {
		panic(fmt.Sprintf("index %d out of bounds for size %d", index, pl.size))
	}
	return pl.lons[index]
}

func (pl *PointList) GetEle(index int) float64 {
	if index >= pl.size {
		panic(fmt.Sprintf("index %d out of bounds for size %d", index, pl.size))
	}
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

func (pl *PointList) Size() int      { return pl.size }
func (pl *PointList) IsEmpty() bool   { return pl.size == 0 }
func (pl *PointList) Is3D() bool      { return pl.is3D }
func (pl *PointList) IsImmutable() bool { return pl.immutable }

func (pl *PointList) MakeImmutable() *PointList {
	pl.immutable = true
	return pl
}

func (pl *PointList) Reverse() {
	pl.ensureMutable()
	for i, j := 0, pl.size-1; i < j; i, j = i+1, j-1 {
		pl.lats[i], pl.lats[j] = pl.lats[j], pl.lats[i]
		pl.lons[i], pl.lons[j] = pl.lons[j], pl.lons[i]
		if pl.is3D {
			pl.eles[i], pl.eles[j] = pl.eles[j], pl.eles[i]
		}
	}
}

func (pl *PointList) Clone(reverse bool) *PointList {
	c := NewPointList(pl.size, pl.is3D)
	for i := 0; i < pl.size; i++ {
		if pl.is3D {
			c.Add3D(pl.lats[i], pl.lons[i], pl.eles[i])
		} else {
			c.Add(pl.lats[i], pl.lons[i])
		}
	}
	if reverse {
		c.Reverse()
	}
	return c
}

func (pl *PointList) Copy(from, end int) *PointList {
	if from > end {
		panic("from must be smaller or equal to end")
	}
	if from < 0 || end > pl.size {
		panic(fmt.Sprintf("illegal interval: %d, %d, size:%d", from, end, pl.size))
	}
	length := end - from
	c := NewPointList(length, pl.is3D)
	c.lats = append(c.lats, pl.lats[from:end]...)
	c.lons = append(c.lons, pl.lons[from:end]...)
	if pl.is3D {
		c.eles = append(c.eles, pl.eles[from:end]...)
	}
	c.size = length
	return c
}

func (pl *PointList) RemoveLastPoint() {
	pl.ensureMutable()
	if pl.size == 0 {
		panic("cannot remove last point from empty PointList")
	}
	pl.size--
	pl.lats = pl.lats[:pl.size]
	pl.lons = pl.lons[:pl.size]
	if pl.is3D {
		pl.eles = pl.eles[:pl.size]
	}
}

func (pl *PointList) TrimToSize(newSize int) {
	pl.ensureMutable()
	if newSize > pl.size {
		panic("new size needs be smaller than old size")
	}
	pl.size = newSize
	pl.lats = pl.lats[:newSize]
	pl.lons = pl.lons[:newSize]
	if pl.is3D {
		pl.eles = pl.eles[:newSize]
	}
}

func (pl *PointList) Clear() {
	pl.ensureMutable()
	pl.size = 0
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
	if pl.size != other.size || pl.is3D != other.is3D {
		return false
	}
	for i := 0; i < pl.size; i++ {
		if !EqualsEps(pl.lats[i], other.lats[i]) {
			return false
		}
		if !EqualsEps(pl.lons[i], other.lons[i]) {
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
	for i := 0; i < pl.size; i++ {
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
	pts := make([]GHPoint, pl.size)
	for i := 0; i < pl.size; i++ {
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
