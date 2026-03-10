package weighting

import (
	"fmt"
	"strings"

	"gohopper/core/config"
	custom_models "gohopper/core/custom_models"
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
	cm := buildCustomModel(profile, vehicleSpeedKey, f.encodingManager)

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

func buildCustomModel(profile config.Profile, vehicleSpeedKey string, em *routingutil.EncodingManager) *webapi.CustomModel {
	// Load custom model files (e.g. car.json) and merge them.
	var cm *webapi.CustomModel
	if len(profile.CustomModelFiles) > 0 {
		cm = webapi.NewCustomModel()
		for _, f := range profile.CustomModelFiles {
			fileCM, err := custom_models.Load(f)
			if err != nil {
				panic(fmt.Sprintf("loading custom model file %q: %v", f, err))
			}
			cm = webapi.MergeCustomModels(cm, fileCM)
		}
	} else {
		// Default: match Java's TestProfiles.accessAndSpeed() — block
		// inaccessible edges via priority and limit speed to vehicle speed.
		accessKey := ev.VehicleAccessKey(profile.Name)
		cm = webapi.NewCustomModel()
		cm.AddToPriority(webapi.If("!"+accessKey, webapi.OpMultiply, "0"))
		cm.AddToSpeed(webapi.If("true", webapi.OpLimit, vehicleSpeedKey))
	}

	// Always block subnetwork edges via priority.
	subnetworkKey := ev.SubnetworkKey(profile.Name)
	if em.HasEncodedValue(subnetworkKey) {
		cm.AddToPriority(webapi.If(subnetworkKey, webapi.OpMultiply, "0"))
	}

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
