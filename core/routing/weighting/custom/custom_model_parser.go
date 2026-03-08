package custom

import (
	"gohopper/core/routing/ev"
	webapi "gohopper/web-api"
)

const (
	globalMaxSpeed        = 999.0
	globalPriority        = 1.0
	defaultHeadingPenalty = 300.0
)

// CreateWeighting builds a CustomWeighting from a CustomModel and encoded value lookup.
func CreateWeighting(lookup ev.EncodedValueLookup, turnCostProvider TurnCostProvider, hasTurnCosts bool, customModel *webapi.CustomModel) *CustomWeighting {
	if customModel == nil {
		panic("CustomModel cannot be nil")
	}
	params := CreateWeightingParameters(customModel, lookup)
	return NewCustomWeighting(turnCostProvider, hasTurnCosts, params)
}

// CreateWeightingParameters computes speed/priority mappings and min/max bounds from a CustomModel.
func CreateWeightingParameters(customModel *webapi.CustomModel, lookup ev.EncodedValueLookup) *Parameters {
	speedMapping := BuildEdgeToDoubleMapping(customModel.Speed, globalMaxSpeed, lookup)
	priorityMapping := BuildEdgeToDoubleMapping(customModel.Priority, globalPriority, lookup)

	// Compute max speed
	speedMinMax := webapi.MinMax{Min: 0, Max: globalMaxSpeed}
	FindMinMax(&speedMinMax, customModel.Speed, lookup)
	maxSpeed := speedMinMax.Max

	// Compute max priority
	prioMinMax := webapi.MinMax{Min: 0, Max: globalPriority}
	FindMinMax(&prioMinMax, customModel.Priority, lookup)
	maxPriority := prioMinMax.Max

	// Clamp to reasonable bounds.
	maxSpeed = max(min(maxSpeed, globalMaxSpeed), 1)
	maxPriority = max(maxPriority, 0.01)

	di := 0.0
	if customModel.DistanceInfluence != nil {
		di = *customModel.DistanceInfluence
	}

	hp := defaultHeadingPenalty
	if customModel.HeadingPenalty != nil {
		hp = *customModel.HeadingPenalty
	}

	return &Parameters{
		EdgeToSpeed:       speedMapping,
		EdgeToPriority:    priorityMapping,
		MaxSpeed:          maxSpeed,
		MaxPriority:       maxPriority,
		DistanceInfluence: di,
		HeadingPenalty:    hp,
	}
}
