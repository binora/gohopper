package weighting

import (
	"math"
	"testing"

	"gohopper/core/routing/ev"
	"gohopper/core/util"
)

// --- mock EdgeIteratorState for tests ---

type mockEdgeIteratorState struct {
	distance float64
	speed    float64
	revSpeed float64
}

func (m *mockEdgeIteratorState) GetDistance() float64                                           { return m.distance }
func (m *mockEdgeIteratorState) GetDecimal(ev.DecimalEncodedValue) float64                     { return m.speed }
func (m *mockEdgeIteratorState) GetReverseDecimal(ev.DecimalEncodedValue) float64              { return m.revSpeed }
func (m *mockEdgeIteratorState) GetEdge() int                                                  { return 0 }
func (m *mockEdgeIteratorState) GetEdgeKey() int                                               { return 0 }
func (m *mockEdgeIteratorState) GetReverseEdgeKey() int                                        { return 0 }
func (m *mockEdgeIteratorState) GetBaseNode() int                                              { return 0 }
func (m *mockEdgeIteratorState) GetAdjNode() int                                               { return 0 }
func (m *mockEdgeIteratorState) FetchWayGeometry(util.FetchMode) *util.PointList               { return nil }
func (m *mockEdgeIteratorState) SetWayGeometry(*util.PointList) util.EdgeIteratorState          { return m }
func (m *mockEdgeIteratorState) SetDistance(float64) util.EdgeIteratorState                     { return m }
func (m *mockEdgeIteratorState) GetBool(ev.BooleanEncodedValue) bool                           { return false }
func (m *mockEdgeIteratorState) SetBool(ev.BooleanEncodedValue, bool) util.EdgeIteratorState    { return m }
func (m *mockEdgeIteratorState) GetReverseBool(ev.BooleanEncodedValue) bool                    { return false }
func (m *mockEdgeIteratorState) SetReverseBool(ev.BooleanEncodedValue, bool) util.EdgeIteratorState { return m }
func (m *mockEdgeIteratorState) SetBoolBothDir(ev.BooleanEncodedValue, bool, bool) util.EdgeIteratorState {
	return m
}
func (m *mockEdgeIteratorState) GetInt(ev.IntEncodedValue) int32                                      { return 0 }
func (m *mockEdgeIteratorState) SetInt(ev.IntEncodedValue, int32) util.EdgeIteratorState               { return m }
func (m *mockEdgeIteratorState) GetReverseInt(ev.IntEncodedValue) int32                                { return 0 }
func (m *mockEdgeIteratorState) SetReverseInt(ev.IntEncodedValue, int32) util.EdgeIteratorState        { return m }
func (m *mockEdgeIteratorState) SetIntBothDir(ev.IntEncodedValue, int32, int32) util.EdgeIteratorState { return m }
func (m *mockEdgeIteratorState) SetDecimal(ev.DecimalEncodedValue, float64) util.EdgeIteratorState     { return m }
func (m *mockEdgeIteratorState) SetReverseDecimal(ev.DecimalEncodedValue, float64) util.EdgeIteratorState {
	return m
}
func (m *mockEdgeIteratorState) SetDecimalBothDir(ev.DecimalEncodedValue, float64, float64) util.EdgeIteratorState {
	return m
}
func (m *mockEdgeIteratorState) GetEnum(any) any                                                    { return nil }
func (m *mockEdgeIteratorState) SetEnum(any, any) util.EdgeIteratorState                            { return m }
func (m *mockEdgeIteratorState) GetReverseEnum(any) any                                             { return nil }
func (m *mockEdgeIteratorState) SetReverseEnum(any, any) util.EdgeIteratorState                     { return m }
func (m *mockEdgeIteratorState) SetEnumBothDir(any, any, any) util.EdgeIteratorState                { return m }
func (m *mockEdgeIteratorState) GetString(*ev.StringEncodedValue) string                            { return "" }
func (m *mockEdgeIteratorState) SetString(*ev.StringEncodedValue, string) util.EdgeIteratorState     { return m }
func (m *mockEdgeIteratorState) GetReverseString(*ev.StringEncodedValue) string                     { return "" }
func (m *mockEdgeIteratorState) SetReverseString(*ev.StringEncodedValue, string) util.EdgeIteratorState {
	return m
}
func (m *mockEdgeIteratorState) SetStringBothDir(*ev.StringEncodedValue, string, string) util.EdgeIteratorState {
	return m
}
func (m *mockEdgeIteratorState) GetName() string                                          { return "" }
func (m *mockEdgeIteratorState) SetKeyValues(map[string]any) util.EdgeIteratorState       { return m }
func (m *mockEdgeIteratorState) GetKeyValues() map[string]any                             { return nil }
func (m *mockEdgeIteratorState) GetValue(string) any                                      { return nil }
func (m *mockEdgeIteratorState) Detach(bool) util.EdgeIteratorState                       { return m }
func (m *mockEdgeIteratorState) CopyPropertiesFrom(util.EdgeIteratorState) util.EdgeIteratorState { return m }

// --- mock DecimalEncodedValue for tests ---

type mockDecimalEV struct {
	maxStorable float64
}

func (m *mockDecimalEV) Init(*ev.InitializerConfig) int                                 { return 0 }
func (m *mockDecimalEV) GetName() string                                                { return "speed" }
func (m *mockDecimalEV) IsStoreTwoDirections() bool                                     { return true }
func (m *mockDecimalEV) SetDecimal(bool, int, ev.EdgeIntAccess, float64)                {}
func (m *mockDecimalEV) GetDecimal(bool, int, ev.EdgeIntAccess) float64                 { return 0 }
func (m *mockDecimalEV) GetMaxStorableDecimal() float64                                 { return m.maxStorable }
func (m *mockDecimalEV) GetMinStorableDecimal() float64                                 { return 0 }
func (m *mockDecimalEV) GetMaxOrMaxStorableDecimal() float64                            { return m.maxStorable }
func (m *mockDecimalEV) GetNextStorableValue(v float64) float64                         { return v }
func (m *mockDecimalEV) GetSmallestNonZeroValue() float64                               { return 1 }

// --- tests ---

// TestIsValidName mirrors WeightingTest.testToString from Java GraphHopper.
func TestIsValidName(t *testing.T) {
	if !IsValidName("blup") {
		t.Error("expected 'blup' to be valid")
	}
	if !IsValidName("blup_a") {
		t.Error("expected 'blup_a' to be valid")
	}
	if !IsValidName("blup|a") {
		t.Error("expected 'blup|a' to be valid")
	}
	if IsValidName("Blup") {
		t.Error("expected 'Blup' to be invalid")
	}
	if IsValidName("Blup!") {
		t.Error("expected 'Blup!' to be invalid")
	}
}

func TestSpeedWeighting_ZeroSpeed(t *testing.T) {
	enc := &mockDecimalEV{maxStorable: 100}
	w := NewSpeedWeighting(enc)

	edge := &mockEdgeIteratorState{distance: 1000, speed: 0, revSpeed: 0}
	weight := w.CalcEdgeWeight(edge, false)
	if !math.IsInf(weight, 1) {
		t.Errorf("expected +Inf for zero speed, got %f", weight)
	}
}

func TestSpeedWeighting_PositiveSpeed(t *testing.T) {
	enc := &mockDecimalEV{maxStorable: 100}
	w := NewSpeedWeighting(enc)

	edge := &mockEdgeIteratorState{distance: 1000, speed: 50, revSpeed: 25}

	weight := w.CalcEdgeWeight(edge, false)
	expected := 1000.0 / 50.0
	if math.Abs(weight-expected) > 1e-9 {
		t.Errorf("expected %f, got %f", expected, weight)
	}

	revWeight := w.CalcEdgeWeight(edge, true)
	expectedRev := 1000.0 / 25.0
	if math.Abs(revWeight-expectedRev) > 1e-9 {
		t.Errorf("expected %f for reverse, got %f", expectedRev, revWeight)
	}

	millis := w.CalcEdgeMillis(edge, false)
	expectedMillis := int64(1000 * expected)
	if millis != expectedMillis {
		t.Errorf("expected %d millis, got %d", expectedMillis, millis)
	}
}

func TestSpeedWeighting_NoTurnCosts(t *testing.T) {
	enc := &mockDecimalEV{maxStorable: 100}
	w := NewSpeedWeighting(enc)

	if w.HasTurnCosts() {
		t.Error("expected HasTurnCosts to be false for NoTurnCostProvider")
	}
	if w.CalcTurnWeight(0, 1, 2) != 0 {
		t.Error("expected zero turn weight")
	}
	if w.CalcTurnMillis(0, 1, 2) != 0 {
		t.Error("expected zero turn millis")
	}
}

func TestSpeedWeighting_WithTurnCostProvider(t *testing.T) {
	enc := &mockDecimalEV{maxStorable: 100}
	tcp := &mockTurnCostProvider{weight: 5.0, millis: 5000}
	w := NewSpeedWeightingWithProvider(enc, tcp)

	if !w.HasTurnCosts() {
		t.Error("expected HasTurnCosts to be true with custom provider")
	}
	if w.CalcTurnWeight(0, 1, 2) != 5.0 {
		t.Errorf("expected turn weight 5.0, got %f", w.CalcTurnWeight(0, 1, 2))
	}
	if w.CalcTurnMillis(0, 1, 2) != 5000 {
		t.Errorf("expected turn millis 5000, got %d", w.CalcTurnMillis(0, 1, 2))
	}
}

func TestSpeedWeighting_CalcMinWeightPerDistance(t *testing.T) {
	enc := &mockDecimalEV{maxStorable: 200}
	w := NewSpeedWeighting(enc)

	expected := 1.0 / 200.0
	got := w.CalcMinWeightPerDistance()
	if math.Abs(got-expected) > 1e-12 {
		t.Errorf("expected %f, got %f", expected, got)
	}
}

func TestSpeedWeighting_GetName(t *testing.T) {
	enc := &mockDecimalEV{maxStorable: 100}
	w := NewSpeedWeighting(enc)

	if w.GetName() != "speed" {
		t.Errorf("expected name 'speed', got %q", w.GetName())
	}
}

// --- mock TurnCostProvider ---

type mockTurnCostProvider struct {
	weight float64
	millis int64
}

func (m *mockTurnCostProvider) CalcTurnWeight(int, int, int) float64 { return m.weight }
func (m *mockTurnCostProvider) CalcTurnMillis(int, int, int) int64   { return m.millis }
