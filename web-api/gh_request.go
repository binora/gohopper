package webapi

import (
	"math"

	"gohopper/core/util"
)

type GHRequest struct {
	Points          []util.GHPoint    `json:"points"`
	Profile         string            `json:"profile,omitempty"`
	Algorithm       string            `json:"algorithm,omitempty"`
	Locale          string            `json:"locale,omitempty"`
	Headings        []float64         `json:"headings,omitempty"`
	PointHints      []string          `json:"point_hints,omitempty"`
	Curbsides       []string          `json:"curbsides,omitempty"`
	SnapPreventions []string          `json:"snap_preventions,omitempty"`
	PathDetails     []string          `json:"details,omitempty"`
	Hints           PMap              `json:"hints,omitempty"`
	CustomModel     map[string]any    `json:"custom_model,omitempty"`
	RawCustomModel  map[string]any    `json:"-"`
	Options         RouteRequestFlags `json:"-"`
}

type RouteRequestFlags struct {
	OutputType              string
	Instructions            bool
	CalcPoints              bool
	Elevation               bool
	PointsEncoded           bool
	PointsEncodedMultiplier float64
	WithRoute               bool
	WithTrack               bool
	WithWayPoints           bool
	TrackName               string
	GPXMillis               string
}

func NewGHRequest() GHRequest {
	return GHRequest{
		Hints: NewPMap(),
		Options: RouteRequestFlags{
			OutputType:              "json",
			Instructions:            true,
			CalcPoints:              true,
			PointsEncoded:           true,
			PointsEncodedMultiplier: 1e5,
			WithRoute:               true,
			WithTrack:               true,
			WithWayPoints:           false,
			TrackName:               "GraphHopper Track",
		},
	}
}

func IsAzimuthValue(v float64) bool {
	if math.IsNaN(v) {
		return true
	}
	return v >= 0 && v < 360
}

func (r GHRequest) HasSnapPreventions() bool {
	return r.SnapPreventions != nil
}
