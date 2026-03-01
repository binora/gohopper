package weighting

import (
	"math"
	"testing"
)

type mockEdgeWithID struct {
	mockEdgeIteratorState
	edgeID int
}

func (m *mockEdgeWithID) GetEdge() int { return m.edgeID }

func TestAvoidEdgesWeighting_PenaltyApplied(t *testing.T) {
	enc := &mockDecimalEV{maxStorable: 100}
	super := NewSpeedWeighting(enc)

	avoided := map[int]struct{}{42: {}}
	w := NewAvoidEdgesWeighting(super).SetAvoidedEdges(avoided)

	edge := &mockEdgeWithID{
		mockEdgeIteratorState: mockEdgeIteratorState{distance: 1000, speed: 50},
		edgeID:                42,
	}

	baseWeight := super.CalcEdgeWeight(edge, false)
	penalizedWeight := w.CalcEdgeWeight(edge, false)

	expected := baseWeight * 5.0
	if math.Abs(penalizedWeight-expected) > 1e-9 {
		t.Errorf("expected penalized weight %f, got %f", expected, penalizedWeight)
	}
}

func TestAvoidEdgesWeighting_NoPenaltyOnNonAvoided(t *testing.T) {
	enc := &mockDecimalEV{maxStorable: 100}
	super := NewSpeedWeighting(enc)

	avoided := map[int]struct{}{42: {}}
	w := NewAvoidEdgesWeighting(super).SetAvoidedEdges(avoided)

	edge := &mockEdgeWithID{
		mockEdgeIteratorState: mockEdgeIteratorState{distance: 1000, speed: 50},
		edgeID:                99,
	}

	baseWeight := super.CalcEdgeWeight(edge, false)
	weight := w.CalcEdgeWeight(edge, false)

	if math.Abs(weight-baseWeight) > 1e-9 {
		t.Errorf("expected weight %f (no penalty), got %f", baseWeight, weight)
	}
}

func TestAvoidEdgesWeighting_CustomPenaltyFactor(t *testing.T) {
	enc := &mockDecimalEV{maxStorable: 100}
	super := NewSpeedWeighting(enc)

	avoided := map[int]struct{}{7: {}}
	w := NewAvoidEdgesWeighting(super).
		SetAvoidedEdges(avoided).
		SetEdgePenaltyFactor(10.0)

	edge := &mockEdgeWithID{
		mockEdgeIteratorState: mockEdgeIteratorState{distance: 500, speed: 25},
		edgeID:                7,
	}

	baseWeight := super.CalcEdgeWeight(edge, false)
	penalizedWeight := w.CalcEdgeWeight(edge, false)

	expected := baseWeight * 10.0
	if math.Abs(penalizedWeight-expected) > 1e-9 {
		t.Errorf("expected penalized weight %f, got %f", expected, penalizedWeight)
	}
}

func TestAvoidEdgesWeighting_DelegatesOtherMethods(t *testing.T) {
	enc := &mockDecimalEV{maxStorable: 200}
	tcp := &mockTurnCostProvider{weight: 3.0, millis: 3000}
	super := NewSpeedWeightingWithProvider(enc, tcp)

	w := NewAvoidEdgesWeighting(super)

	if w.CalcMinWeightPerDistance() != super.CalcMinWeightPerDistance() {
		t.Error("CalcMinWeightPerDistance not delegated")
	}
	if w.CalcTurnWeight(0, 1, 2) != 3.0 {
		t.Error("CalcTurnWeight not delegated")
	}
	if w.CalcTurnMillis(0, 1, 2) != 3000 {
		t.Error("CalcTurnMillis not delegated")
	}
	if !w.HasTurnCosts() {
		t.Error("HasTurnCosts not delegated")
	}
}

func TestAvoidEdgesWeighting_GetName(t *testing.T) {
	enc := &mockDecimalEV{maxStorable: 100}
	w := NewAvoidEdgesWeighting(NewSpeedWeighting(enc))

	if w.GetName() != "avoid_edges" {
		t.Errorf("expected name 'avoid_edges', got %q", w.GetName())
	}
}

func TestAvoidEdgesWeighting_NilSuperPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil super weighting")
		}
	}()
	NewAvoidEdgesWeighting(nil)
}
