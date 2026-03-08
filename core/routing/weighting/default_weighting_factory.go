package weighting

import (
	"fmt"
	"strings"

	"gohopper/core/config"
	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting/custom"
	"gohopper/core/storage"
	webapi "gohopper/web-api"
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

	vehicleSpeedKey := ev.VehicleSpeedKey(profile.Name)

	// Build CustomModel: use profile's model or create default
	cm := buildCustomModel(profile, vehicleSpeedKey)

	hasTurnCosts := false
	var tcp custom.TurnCostProvider = NoTurnCostProvider
	if profile.TurnCosts != nil && !disableTurnCosts {
		hasTurnCosts = true
		turnRestrictionEnc := f.encodingManager.GetTurnBooleanEncodedValue(ev.TurnRestrictionKey(profile.Name))

		uTurnCosts := -1 // infinite by default
		if v, ok := profile.TurnCosts["u_turn_costs"]; ok {
			uTurnCosts = toInt(v)
		}
		if v, ok := hints["u_turn_costs"]; ok {
			uTurnCosts = toInt(v)
		}

		tcp = NewDefaultTurnCostProvider(
			turnRestrictionEnc,
			f.graph.TurnCostStorage,
			f.graph.GetNodeAccess(),
			uTurnCosts,
		)
	}

	return custom.CreateWeighting(f.encodingManager, tcp, hasTurnCosts, cm)
}

func buildCustomModel(profile config.Profile, vehicleSpeedKey string) *webapi.CustomModel {
	// TODO: parse profile.CustomModel map[string]any into *webapi.CustomModel
	// Default: block inaccessible edges and limit speed to vehicle speed.
	// In Go, speed parsers set speed for both directions, so we must
	// explicitly check access (Java parsers set speed=0 for inaccessible dirs).
	accessKey := ev.VehicleAccessKey(profile.Name)
	cm := webapi.NewCustomModel()
	cm.AddToSpeed(webapi.If("!"+accessKey, webapi.OpMultiply, "0"))
	cm.AddToSpeed(webapi.If("true", webapi.OpLimit, vehicleSpeedKey))
	return cm
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
