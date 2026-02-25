package util

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const earthRadiusMeters = 6371000.0

type GHPoint struct {
	Lat float64 `json:"-"`
	Lon float64 `json:"-"`
}

func ParseGHPoint(value string) (GHPoint, error) {
	parts := strings.Split(strings.TrimSpace(value), ",")
	if len(parts) != 2 {
		return GHPoint{}, fmt.Errorf("point must be in 'lat,lon' format")
	}
	lat, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return GHPoint{}, fmt.Errorf("invalid latitude: %w", err)
	}
	lon, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return GHPoint{}, fmt.Errorf("invalid longitude: %w", err)
	}
	return GHPoint{Lat: lat, Lon: lon}, nil
}

func ParseLonLat(values []float64) (GHPoint, error) {
	if len(values) < 2 {
		return GHPoint{}, fmt.Errorf("point must be [lon,lat]")
	}
	return GHPoint{Lat: values[1], Lon: values[0]}, nil
}

func (p GHPoint) String() string {
	return fmt.Sprintf("%f,%f", p.Lat, p.Lon)
}

func DistanceCalcEarth(a, b GHPoint) float64 {
	lat1 := a.Lat * math.Pi / 180
	lat2 := b.Lat * math.Pi / 180
	dLat := (b.Lat - a.Lat) * math.Pi / 180
	dLon := (b.Lon - a.Lon) * math.Pi / 180
	s1 := math.Sin(dLat / 2)
	s2 := math.Sin(dLon / 2)
	h := s1*s1 + math.Cos(lat1)*math.Cos(lat2)*s2*s2
	return 2 * earthRadiusMeters * math.Asin(math.Sqrt(h))
}

func CalcBBox(points []GHPoint) [4]float64 {
	if len(points) == 0 {
		return [4]float64{}
	}
	minLon, minLat := points[0].Lon, points[0].Lat
	maxLon, maxLat := points[0].Lon, points[0].Lat
	for i := 1; i < len(points); i++ {
		p := points[i]
		if p.Lon < minLon {
			minLon = p.Lon
		}
		if p.Lon > maxLon {
			maxLon = p.Lon
		}
		if p.Lat < minLat {
			minLat = p.Lat
		}
		if p.Lat > maxLat {
			maxLat = p.Lat
		}
	}
	return [4]float64{minLon, minLat, maxLon, maxLat}
}
