package util

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// GHPoint represents a geographic point with latitude and longitude.
type GHPoint struct {
	Lat float64 `json:"-"`
	Lon float64 `json:"-"`
}

// GHPoint3D extends GHPoint with elevation.
type GHPoint3D struct {
	GHPoint
	Ele float64
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

func (p GHPoint) IsValid() bool {
	return !math.IsNaN(p.Lat) && !math.IsNaN(p.Lon)
}

// HaversineDistance computes the haversine distance between two points in meters.
// This is a convenience wrapper for DistEarth.CalcDist.
func HaversineDistance(a, b GHPoint) float64 {
	return DistEarth.CalcDist(a.Lat, a.Lon, b.Lat, b.Lon)
}
