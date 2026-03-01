package routing

import (
	"math"
	"testing"

	"gohopper/core/routing/ev"
	"gohopper/core/util"
)

// --- mock EdgeIteratorState for finite-weight tests ---

type fwfMockEdgeIteratorState struct {
	util.EdgeIteratorState
}

func (m *fwfMockEdgeIteratorState) GetEdge() int { return 0 }

// --- mock Weighting for finite-weight tests ---

type fwfMockWeighting struct {
	fwdWeight float64
	bwdWeight float64
}

func (m *fwfMockWeighting) CalcMinWeightPerDistance() float64                             { return 1.0 }
func (m *fwfMockWeighting) CalcEdgeWeight(_ util.EdgeIteratorState, reverse bool) float64 {
	if reverse {
		return m.bwdWeight
	}
	return m.fwdWeight
}
func (m *fwfMockWeighting) CalcEdgeMillis(_ util.EdgeIteratorState, _ bool) int64 { return 0 }
func (m *fwfMockWeighting) CalcTurnWeight(_, _, _ int) float64                    { return 0 }
func (m *fwfMockWeighting) CalcTurnMillis(_, _, _ int) int64                      { return 0 }
func (m *fwfMockWeighting) HasTurnCosts() bool                                    { return false }
func (m *fwfMockWeighting) GetName() string                                       { return "mock" }

// --- mock full EdgeIteratorState ---

type fwfFullMockEdgeIteratorState struct{}

func (m *fwfFullMockEdgeIteratorState) GetEdge() int                                                              { return 0 }
func (m *fwfFullMockEdgeIteratorState) GetEdgeKey() int                                                           { return 0 }
func (m *fwfFullMockEdgeIteratorState) GetReverseEdgeKey() int                                                    { return 0 }
func (m *fwfFullMockEdgeIteratorState) GetBaseNode() int                                                          { return 0 }
func (m *fwfFullMockEdgeIteratorState) GetAdjNode() int                                                           { return 0 }
func (m *fwfFullMockEdgeIteratorState) FetchWayGeometry(util.FetchMode) *util.PointList                           { return nil }
func (m *fwfFullMockEdgeIteratorState) SetWayGeometry(*util.PointList) util.EdgeIteratorState                     { return m }
func (m *fwfFullMockEdgeIteratorState) GetDistance() float64                                                      { return 0 }
func (m *fwfFullMockEdgeIteratorState) SetDistance(float64) util.EdgeIteratorState                                { return m }
func (m *fwfFullMockEdgeIteratorState) GetBool(ev.BooleanEncodedValue) bool                                      { return false }
func (m *fwfFullMockEdgeIteratorState) SetBool(ev.BooleanEncodedValue, bool) util.EdgeIteratorState               { return m }
func (m *fwfFullMockEdgeIteratorState) GetReverseBool(ev.BooleanEncodedValue) bool                               { return false }
func (m *fwfFullMockEdgeIteratorState) SetReverseBool(ev.BooleanEncodedValue, bool) util.EdgeIteratorState        { return m }
func (m *fwfFullMockEdgeIteratorState) SetBoolBothDir(ev.BooleanEncodedValue, bool, bool) util.EdgeIteratorState  { return m }
func (m *fwfFullMockEdgeIteratorState) GetInt(ev.IntEncodedValue) int32                                           { return 0 }
func (m *fwfFullMockEdgeIteratorState) SetInt(ev.IntEncodedValue, int32) util.EdgeIteratorState                   { return m }
func (m *fwfFullMockEdgeIteratorState) GetReverseInt(ev.IntEncodedValue) int32                                    { return 0 }
func (m *fwfFullMockEdgeIteratorState) SetReverseInt(ev.IntEncodedValue, int32) util.EdgeIteratorState            { return m }
func (m *fwfFullMockEdgeIteratorState) SetIntBothDir(ev.IntEncodedValue, int32, int32) util.EdgeIteratorState     { return m }
func (m *fwfFullMockEdgeIteratorState) GetDecimal(ev.DecimalEncodedValue) float64                                 { return 0 }
func (m *fwfFullMockEdgeIteratorState) SetDecimal(ev.DecimalEncodedValue, float64) util.EdgeIteratorState         { return m }
func (m *fwfFullMockEdgeIteratorState) GetReverseDecimal(ev.DecimalEncodedValue) float64                          { return 0 }
func (m *fwfFullMockEdgeIteratorState) SetReverseDecimal(ev.DecimalEncodedValue, float64) util.EdgeIteratorState  { return m }
func (m *fwfFullMockEdgeIteratorState) SetDecimalBothDir(ev.DecimalEncodedValue, float64, float64) util.EdgeIteratorState {
	return m
}
func (m *fwfFullMockEdgeIteratorState) GetEnum(any) any                                                        { return nil }
func (m *fwfFullMockEdgeIteratorState) SetEnum(any, any) util.EdgeIteratorState                                { return m }
func (m *fwfFullMockEdgeIteratorState) GetReverseEnum(any) any                                                 { return nil }
func (m *fwfFullMockEdgeIteratorState) SetReverseEnum(any, any) util.EdgeIteratorState                         { return m }
func (m *fwfFullMockEdgeIteratorState) SetEnumBothDir(any, any, any) util.EdgeIteratorState                    { return m }
func (m *fwfFullMockEdgeIteratorState) GetString(*ev.StringEncodedValue) string                                { return "" }
func (m *fwfFullMockEdgeIteratorState) SetString(*ev.StringEncodedValue, string) util.EdgeIteratorState        { return m }
func (m *fwfFullMockEdgeIteratorState) GetReverseString(*ev.StringEncodedValue) string                         { return "" }
func (m *fwfFullMockEdgeIteratorState) SetReverseString(*ev.StringEncodedValue, string) util.EdgeIteratorState { return m }
func (m *fwfFullMockEdgeIteratorState) SetStringBothDir(*ev.StringEncodedValue, string, string) util.EdgeIteratorState {
	return m
}
func (m *fwfFullMockEdgeIteratorState) GetName() string                                                  { return "" }
func (m *fwfFullMockEdgeIteratorState) SetKeyValues(map[string]any) util.EdgeIteratorState               { return m }
func (m *fwfFullMockEdgeIteratorState) GetKeyValues() map[string]any                                     { return nil }
func (m *fwfFullMockEdgeIteratorState) GetValue(string) any                                              { return nil }
func (m *fwfFullMockEdgeIteratorState) Detach(bool) util.EdgeIteratorState                               { return m }
func (m *fwfFullMockEdgeIteratorState) CopyPropertiesFrom(util.EdgeIteratorState) util.EdgeIteratorState { return m }

// --- tests ---

func TestFiniteWeightFilter_Accept(t *testing.T) {
	w := &fwfMockWeighting{fwdWeight: 10.0, bwdWeight: 20.0}
	filter := NewFiniteWeightFilter(w)
	edge := &fwfFullMockEdgeIteratorState{}

	if !filter(edge) {
		t.Error("expected finite weight edge to be accepted")
	}
}

func TestFiniteWeightFilter_AcceptOneFwd(t *testing.T) {
	w := &fwfMockWeighting{fwdWeight: 5.0, bwdWeight: math.Inf(1)}
	filter := NewFiniteWeightFilter(w)
	edge := &fwfFullMockEdgeIteratorState{}

	if !filter(edge) {
		t.Error("expected edge with finite forward weight to be accepted")
	}
}

func TestFiniteWeightFilter_AcceptOneBwd(t *testing.T) {
	w := &fwfMockWeighting{fwdWeight: math.Inf(1), bwdWeight: 5.0}
	filter := NewFiniteWeightFilter(w)
	edge := &fwfFullMockEdgeIteratorState{}

	if !filter(edge) {
		t.Error("expected edge with finite backward weight to be accepted")
	}
}

func TestFiniteWeightFilter_Reject(t *testing.T) {
	w := &fwfMockWeighting{fwdWeight: math.Inf(1), bwdWeight: math.Inf(1)}
	filter := NewFiniteWeightFilter(w)
	edge := &fwfFullMockEdgeIteratorState{}

	if filter(edge) {
		t.Error("expected edge with infinite weight in both directions to be rejected")
	}
}

func TestFiniteWeightFilter_RejectNaN(t *testing.T) {
	w := &fwfMockWeighting{fwdWeight: math.NaN(), bwdWeight: math.NaN()}
	filter := NewFiniteWeightFilter(w)
	edge := &fwfFullMockEdgeIteratorState{}

	if filter(edge) {
		t.Error("expected edge with NaN weight in both directions to be rejected")
	}
}
