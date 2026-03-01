package util_test

import (
	"testing"

	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/util"
)

// --- mock BooleanEncodedValue ---

type mockBoolEV struct{ name string }

func (m *mockBoolEV) Init(*ev.InitializerConfig) int                              { return 0 }
func (m *mockBoolEV) GetName() string                                              { return m.name }
func (m *mockBoolEV) IsStoreTwoDirections() bool                                   { return true }
func (m *mockBoolEV) SetBool(bool, int, ev.EdgeIntAccess, bool)                    {}
func (m *mockBoolEV) GetBool(bool, int, ev.EdgeIntAccess) bool                     { return false }

// --- mock EdgeIteratorState ---

type mockEdgeIteratorState struct {
	fwd, bwd bool
	target   ev.BooleanEncodedValue
}

func (m *mockEdgeIteratorState) GetBool(p ev.BooleanEncodedValue) bool {
	if p == m.target {
		return m.fwd
	}
	return false
}
func (m *mockEdgeIteratorState) GetReverseBool(p ev.BooleanEncodedValue) bool {
	if p == m.target {
		return m.bwd
	}
	return false
}

// Remaining interface methods (no-ops for tests).
func (m *mockEdgeIteratorState) GetEdge() int                                                              { return 0 }
func (m *mockEdgeIteratorState) GetEdgeKey() int                                                           { return 0 }
func (m *mockEdgeIteratorState) GetReverseEdgeKey() int                                                    { return 0 }
func (m *mockEdgeIteratorState) GetBaseNode() int                                                          { return 0 }
func (m *mockEdgeIteratorState) GetAdjNode() int                                                           { return 0 }
func (m *mockEdgeIteratorState) FetchWayGeometry(util.FetchMode) *util.PointList                           { return nil }
func (m *mockEdgeIteratorState) SetWayGeometry(*util.PointList) util.EdgeIteratorState                     { return m }
func (m *mockEdgeIteratorState) GetDistance() float64                                                      { return 0 }
func (m *mockEdgeIteratorState) SetDistance(float64) util.EdgeIteratorState                                { return m }
func (m *mockEdgeIteratorState) SetBool(ev.BooleanEncodedValue, bool) util.EdgeIteratorState               { return m }
func (m *mockEdgeIteratorState) SetReverseBool(ev.BooleanEncodedValue, bool) util.EdgeIteratorState        { return m }
func (m *mockEdgeIteratorState) SetBoolBothDir(ev.BooleanEncodedValue, bool, bool) util.EdgeIteratorState  { return m }
func (m *mockEdgeIteratorState) GetInt(ev.IntEncodedValue) int32                                           { return 0 }
func (m *mockEdgeIteratorState) SetInt(ev.IntEncodedValue, int32) util.EdgeIteratorState                   { return m }
func (m *mockEdgeIteratorState) GetReverseInt(ev.IntEncodedValue) int32                                    { return 0 }
func (m *mockEdgeIteratorState) SetReverseInt(ev.IntEncodedValue, int32) util.EdgeIteratorState            { return m }
func (m *mockEdgeIteratorState) SetIntBothDir(ev.IntEncodedValue, int32, int32) util.EdgeIteratorState     { return m }
func (m *mockEdgeIteratorState) GetDecimal(ev.DecimalEncodedValue) float64                                 { return 0 }
func (m *mockEdgeIteratorState) SetDecimal(ev.DecimalEncodedValue, float64) util.EdgeIteratorState         { return m }
func (m *mockEdgeIteratorState) GetReverseDecimal(ev.DecimalEncodedValue) float64                          { return 0 }
func (m *mockEdgeIteratorState) SetReverseDecimal(ev.DecimalEncodedValue, float64) util.EdgeIteratorState  { return m }
func (m *mockEdgeIteratorState) SetDecimalBothDir(ev.DecimalEncodedValue, float64, float64) util.EdgeIteratorState {
	return m
}
func (m *mockEdgeIteratorState) GetEnum(any) any                                                        { return nil }
func (m *mockEdgeIteratorState) SetEnum(any, any) util.EdgeIteratorState                                { return m }
func (m *mockEdgeIteratorState) GetReverseEnum(any) any                                                 { return nil }
func (m *mockEdgeIteratorState) SetReverseEnum(any, any) util.EdgeIteratorState                         { return m }
func (m *mockEdgeIteratorState) SetEnumBothDir(any, any, any) util.EdgeIteratorState                    { return m }
func (m *mockEdgeIteratorState) GetString(*ev.StringEncodedValue) string                                { return "" }
func (m *mockEdgeIteratorState) SetString(*ev.StringEncodedValue, string) util.EdgeIteratorState        { return m }
func (m *mockEdgeIteratorState) GetReverseString(*ev.StringEncodedValue) string                         { return "" }
func (m *mockEdgeIteratorState) SetReverseString(*ev.StringEncodedValue, string) util.EdgeIteratorState { return m }
func (m *mockEdgeIteratorState) SetStringBothDir(*ev.StringEncodedValue, string, string) util.EdgeIteratorState {
	return m
}
func (m *mockEdgeIteratorState) GetName() string                                                  { return "" }
func (m *mockEdgeIteratorState) SetKeyValues(map[string]any) util.EdgeIteratorState               { return m }
func (m *mockEdgeIteratorState) GetKeyValues() map[string]any                                     { return nil }
func (m *mockEdgeIteratorState) GetValue(string) any                                              { return nil }
func (m *mockEdgeIteratorState) Detach(bool) util.EdgeIteratorState                               { return m }
func (m *mockEdgeIteratorState) CopyPropertiesFrom(util.EdgeIteratorState) util.EdgeIteratorState { return m }

// --- tests ---

func TestAccessFilter_OutEdges(t *testing.T) {
	enc := &mockBoolEV{name: "car_access"}

	// Forward-only edge: accepted by OutEdges.
	edge := &mockEdgeIteratorState{fwd: true, bwd: false, target: enc}
	f := routingutil.OutEdges(enc)
	if !f.Accept(edge) {
		t.Error("OutEdges should accept forward-accessible edge")
	}

	// Backward-only edge: rejected by OutEdges.
	edge2 := &mockEdgeIteratorState{fwd: false, bwd: true, target: enc}
	if f.Accept(edge2) {
		t.Error("OutEdges should reject backward-only edge")
	}
}

func TestAccessFilter_InEdges(t *testing.T) {
	enc := &mockBoolEV{name: "car_access"}

	// Backward-only edge: accepted by InEdges.
	edge := &mockEdgeIteratorState{fwd: false, bwd: true, target: enc}
	f := routingutil.InEdges(enc)
	if !f.Accept(edge) {
		t.Error("InEdges should accept backward-accessible edge")
	}

	// Forward-only edge: rejected by InEdges.
	edge2 := &mockEdgeIteratorState{fwd: true, bwd: false, target: enc}
	if f.Accept(edge2) {
		t.Error("InEdges should reject forward-only edge")
	}
}

func TestAccessFilter_AllEdges(t *testing.T) {
	enc := &mockBoolEV{name: "car_access"}
	f := routingutil.AllAccessEdges(enc)

	fwdOnly := &mockEdgeIteratorState{fwd: true, bwd: false, target: enc}
	if !f.Accept(fwdOnly) {
		t.Error("AllAccessEdges should accept forward-only edge")
	}

	bwdOnly := &mockEdgeIteratorState{fwd: false, bwd: true, target: enc}
	if !f.Accept(bwdOnly) {
		t.Error("AllAccessEdges should accept backward-only edge")
	}

	both := &mockEdgeIteratorState{fwd: true, bwd: true, target: enc}
	if !f.Accept(both) {
		t.Error("AllAccessEdges should accept bidirectional edge")
	}
}

func TestAccessFilter_NeitherDirection(t *testing.T) {
	enc := &mockBoolEV{name: "car_access"}
	edge := &mockEdgeIteratorState{fwd: false, bwd: false, target: enc}

	if routingutil.OutEdges(enc).Accept(edge) {
		t.Error("OutEdges should reject edge with no access")
	}
	if routingutil.InEdges(enc).Accept(edge) {
		t.Error("InEdges should reject edge with no access")
	}
	if routingutil.AllAccessEdges(enc).Accept(edge) {
		t.Error("AllAccessEdges should reject edge with no access in either direction")
	}
}
