package weighting

import (
	"testing"

	"gohopper/core/config"
	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/storage"
)

// buildTestEM creates an EncodingManager with a speed EV and an optional turn restriction EV.
func buildTestEM(profileName string, withTurnCosts bool) *routingutil.EncodingManager {
	b := routingutil.Start()
	b.Add(ev.VehicleSpeedCreate(profileName, 5, 5, true))
	b.Add(ev.VehicleAccessCreate(profileName))
	if withTurnCosts {
		b.AddTurnCostEncodedValue(ev.TurnRestrictionCreate(profileName))
	}
	return b.Build()
}

// buildTestGraph creates a BaseGraph with node/edge data for factory tests.
func buildTestGraph(em *routingutil.EncodingManager, withTurnCosts bool) *storage.BaseGraph {
	g := storage.NewBaseGraphBuilder(em.BytesForFlags).
		SetWithTurnCosts(withTurnCosts).
		CreateGraph()
	na := g.GetNodeAccess()
	na.SetNode(0, 0, 0, 0)
	na.SetNode(1, 1, 1, 0)
	na.SetNode(2, 2, 2, 0)
	g.Edge(0, 1)
	g.Edge(1, 2)
	return g
}

func TestDefaultWeightingFactory_SpeedWeighting(t *testing.T) {
	em := buildTestEM("car", false)
	g := buildTestGraph(em, false)
	defer g.Close()

	factory := NewDefaultWeightingFactory(g, em)
	profile := config.Profile{
		Name:      "car",
		Weighting: "custom",
	}

	w := factory.CreateWeighting(profile, nil, false)
	if w == nil {
		t.Fatal("expected non-nil weighting")
	}
	if w.GetName() != "custom" {
		t.Errorf("expected name 'custom', got %q", w.GetName())
	}
	if w.HasTurnCosts() {
		t.Error("expected no turn costs")
	}
}

func TestDefaultWeightingFactory_WithTurnCosts(t *testing.T) {
	em := buildTestEM("car", true)
	g := buildTestGraph(em, true)
	defer g.Close()

	factory := NewDefaultWeightingFactory(g, em)
	profile := config.Profile{
		Name:      "car",
		Weighting: "custom",
		TurnCosts: map[string]any{
			"u_turn_costs": 40,
		},
	}

	w := factory.CreateWeighting(profile, nil, false)
	if w == nil {
		t.Fatal("expected non-nil weighting")
	}
	if !w.HasTurnCosts() {
		t.Error("expected turn costs to be enabled")
	}
	if w.GetName() != "custom" {
		t.Errorf("expected name 'custom', got %q", w.GetName())
	}
}

func TestDefaultWeightingFactory_DisableTurnCosts(t *testing.T) {
	em := buildTestEM("car", true)
	g := buildTestGraph(em, true)
	defer g.Close()

	factory := NewDefaultWeightingFactory(g, em)
	profile := config.Profile{
		Name:      "car",
		Weighting: "custom",
		TurnCosts: map[string]any{
			"u_turn_costs": 40,
		},
	}

	// Even though profile has turn costs, disableTurnCosts=true should override.
	w := factory.CreateWeighting(profile, nil, true)
	if w == nil {
		t.Fatal("expected non-nil weighting")
	}
	if w.HasTurnCosts() {
		t.Error("expected turn costs to be disabled when disableTurnCosts is true")
	}
}
