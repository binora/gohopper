package weighting

import (
	"fmt"
	"strings"

	"gohopper/core/config"
	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/storage"
)

// Compile-time check that DefaultWeightingFactory implements WeightingFactory.
var _ WeightingFactory = (*DefaultWeightingFactory)(nil)

// DefaultWeightingFactory creates Weighting instances based on profile configuration.
type DefaultWeightingFactory struct {
	graph           *storage.BaseGraph
	encodingManager *routingutil.EncodingManager
}

func NewDefaultWeightingFactory(graph *storage.BaseGraph, em *routingutil.EncodingManager) *DefaultWeightingFactory {
	return &DefaultWeightingFactory{
		graph:           graph,
		encodingManager: em,
	}
}

func (f *DefaultWeightingFactory) CreateWeighting(profile config.Profile, hints map[string]any, disableTurnCosts bool) Weighting {
	weightingStr := strings.ToLower(profile.Weighting)
	if weightingStr == "" {
		weightingStr = "custom"
	}

	switch weightingStr {
	case "custom":
		// For now, custom weighting maps to SpeedWeighting.
	case "shortest":
		panic("Instead of weighting=shortest use weighting=custom with a high distance_influence")
	case "fastest":
		panic("Instead of weighting=fastest use weighting=custom with a custom model that avoids road_access == DESTINATION")
	case "curvature":
		panic("The curvature weighting is no longer supported since 7.0. Use a custom model with the EncodedValue 'curvature' instead")
	case "short_fastest":
		panic("Instead of weighting=short_fastest use weighting=custom with a distance_influence")
	default:
		panic(fmt.Sprintf("Weighting '%s' not supported", weightingStr))
	}

	speedEnc := f.encodingManager.GetDecimalEncodedValue(ev.VehicleSpeedKey(profile.Name))

	if profile.TurnCosts != nil && !disableTurnCosts {
		turnRestrictionEnc := f.encodingManager.GetTurnBooleanEncodedValue(ev.TurnRestrictionKey(profile.Name))

		uTurnCosts := -1 // infinite by default
		if v, ok := profile.TurnCosts["u_turn_costs"]; ok {
			uTurnCosts = toInt(v)
		}
		if v, ok := hints["u_turn_costs"]; ok {
			uTurnCosts = toInt(v)
		}

		tcp := NewDefaultTurnCostProvider(
			turnRestrictionEnc,
			f.graph.TurnCostStorage,
			f.graph.GetNodeAccess(),
			uTurnCosts,
		)
		return NewSpeedWeightingWithProvider(speedEnc, tcp)
	}

	return NewSpeedWeighting(speedEnc)
}

// toInt converts a value to int, handling common types from JSON/YAML deserialization.
func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case int64:
		return int(n)
	default:
		panic(fmt.Sprintf("cannot convert %T to int", v))
	}
}
