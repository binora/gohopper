package routing

import "math"

type RouterConfig struct {
	MaxVisitedNodes              int
	TimeoutMillis                int64
	MaxRoundTripRetries          int
	NonChMaxWaypointDistance      int
	CalcPoints                   bool
	InstructionsEnabled          bool
	SimplifyResponse             bool
	ElevationWayPointMaxDistance float64
	ActiveLandmarkCount          int
}

func NewRouterConfig() RouterConfig {
	return RouterConfig{
		MaxVisitedNodes:              math.MaxInt,
		TimeoutMillis:                math.MaxInt64,
		MaxRoundTripRetries:          3,
		NonChMaxWaypointDistance:      math.MaxInt,
		CalcPoints:                   true,
		InstructionsEnabled:          true,
		SimplifyResponse:             true,
		ElevationWayPointMaxDistance: math.MaxFloat64,
		ActiveLandmarkCount:          8,
	}
}
