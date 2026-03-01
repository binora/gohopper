package routing

import (
	"math"
	"testing"
)

func TestNewRouterConfig_Defaults(t *testing.T) {
	cfg := NewRouterConfig()

	if cfg.MaxVisitedNodes != math.MaxInt {
		t.Errorf("MaxVisitedNodes = %d, want math.MaxInt", cfg.MaxVisitedNodes)
	}
	if cfg.TimeoutMillis != math.MaxInt64 {
		t.Errorf("TimeoutMillis = %d, want math.MaxInt64", cfg.TimeoutMillis)
	}
	if cfg.MaxRoundTripRetries != 3 {
		t.Errorf("MaxRoundTripRetries = %d, want 3", cfg.MaxRoundTripRetries)
	}
	if cfg.NonChMaxWaypointDistance != math.MaxInt {
		t.Errorf("NonChMaxWaypointDistance = %d, want math.MaxInt", cfg.NonChMaxWaypointDistance)
	}
	if !cfg.CalcPoints {
		t.Error("CalcPoints = false, want true")
	}
	if !cfg.InstructionsEnabled {
		t.Error("InstructionsEnabled = false, want true")
	}
	if !cfg.SimplifyResponse {
		t.Error("SimplifyResponse = false, want true")
	}
	if cfg.ElevationWayPointMaxDistance != math.MaxFloat64 {
		t.Errorf("ElevationWayPointMaxDistance = %g, want math.MaxFloat64", cfg.ElevationWayPointMaxDistance)
	}
	if cfg.ActiveLandmarkCount != 8 {
		t.Errorf("ActiveLandmarkCount = %d, want 8", cfg.ActiveLandmarkCount)
	}
}
