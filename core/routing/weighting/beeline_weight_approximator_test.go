package weighting

import (
	"math"
	"testing"

	"gohopper/core/util"
)

// mockNodeAccess implements storage.NodeAccess for testing.
type mockNodeAccess struct {
	lats []float64
	lons []float64
}

func (m *mockNodeAccess) GetLat(nodeID int) float64        { return m.lats[nodeID] }
func (m *mockNodeAccess) GetLon(nodeID int) float64        { return m.lons[nodeID] }
func (m *mockNodeAccess) GetEle(int) float64               { return 0 }
func (m *mockNodeAccess) SetNode(int, float64, float64, float64) {}
func (m *mockNodeAccess) Is3D() bool                       { return false }
func (m *mockNodeAccess) Dimension() int                   { return 2 }
func (m *mockNodeAccess) EnsureNode(int)                   {}
func (m *mockNodeAccess) GetTurnCostIndex(int) int         { return 0 }
func (m *mockNodeAccess) SetTurnCostIndex(int, int)        {}

type beelineMockWeighting struct {
	minWeightPerDist float64
}

func (w *beelineMockWeighting) CalcMinWeightPerDistance() float64                     { return w.minWeightPerDist }
func (w *beelineMockWeighting) CalcEdgeWeight(util.EdgeIteratorState, bool) float64   { return 0 }
func (w *beelineMockWeighting) CalcEdgeMillis(util.EdgeIteratorState, bool) int64     { return 0 }
func (w *beelineMockWeighting) CalcTurnWeight(int, int, int) float64                  { return 0 }
func (w *beelineMockWeighting) CalcTurnMillis(int, int, int) int64                    { return 0 }
func (w *beelineMockWeighting) HasTurnCosts() bool                                    { return false }
func (w *beelineMockWeighting) GetName() string                                       { return "mock" }

func TestBeelineWeightApproximator_Approximate(t *testing.T) {
	na := &mockNodeAccess{
		lats: []float64{48.0, 49.0},
		lons: []float64{11.0, 12.0},
	}
	w := &beelineMockWeighting{minWeightPerDist: 0.001}
	approx := NewBeelineWeightApproximator(na, w)
	approx.SetTo(1)

	result := approx.Approximate(0)
	dist := util.DistEarth.CalcDist(49.0, 12.0, 48.0, 11.0)
	expected := dist * 0.001

	if math.Abs(result-expected) > 1e-6 {
		t.Errorf("expected %f, got %f", expected, result)
	}
}

func TestBeelineWeightApproximator_SameNode(t *testing.T) {
	na := &mockNodeAccess{
		lats: []float64{48.0},
		lons: []float64{11.0},
	}
	w := &beelineMockWeighting{minWeightPerDist: 0.001}
	approx := NewBeelineWeightApproximator(na, w)
	approx.SetTo(0)

	result := approx.Approximate(0)
	if result != 0 {
		t.Errorf("expected 0 for same node, got %f", result)
	}
}

func TestBeelineWeightApproximator_Epsilon(t *testing.T) {
	na := &mockNodeAccess{
		lats: []float64{48.0, 49.0},
		lons: []float64{11.0, 12.0},
	}
	w := &beelineMockWeighting{minWeightPerDist: 0.001}
	approx := NewBeelineWeightApproximator(na, w)
	approx.SetEpsilon(0.5)
	approx.SetTo(1)

	result := approx.Approximate(0)
	dist := util.DistEarth.CalcDist(49.0, 12.0, 48.0, 11.0)
	expected := dist * 0.001 * 0.5

	if math.Abs(result-expected) > 1e-6 {
		t.Errorf("expected %f, got %f", expected, result)
	}
}

func TestBeelineWeightApproximator_GetSlack(t *testing.T) {
	na := &mockNodeAccess{
		lats: []float64{48.0},
		lons: []float64{11.0},
	}
	w := &beelineMockWeighting{minWeightPerDist: 0.001}
	approx := NewBeelineWeightApproximator(na, w)

	if approx.GetSlack() != 0 {
		t.Errorf("expected slack 0, got %f", approx.GetSlack())
	}
}

func TestBeelineWeightApproximator_Reverse(t *testing.T) {
	na := &mockNodeAccess{
		lats: []float64{48.0, 49.0},
		lons: []float64{11.0, 12.0},
	}
	w := &beelineMockWeighting{minWeightPerDist: 0.001}
	approx := NewBeelineWeightApproximator(na, w)
	approx.SetEpsilon(0.7)

	rev := approx.Reverse()
	bwa, ok := rev.(*BeelineWeightApproximator)
	if !ok {
		t.Fatal("Reverse() should return *BeelineWeightApproximator")
	}

	if bwa.epsilon != 0.7 {
		t.Errorf("expected epsilon 0.7 in reverse, got %f", bwa.epsilon)
	}

	// The reversed approximator must be independent: setting to on one
	// must not affect the other.
	approx.SetTo(1)
	rev.SetTo(0)

	if approx.toLat != na.lats[1] || approx.toLon != na.lons[1] {
		t.Error("original toLat/toLon should not be affected by reverse SetTo")
	}
	if bwa.toLat != na.lats[0] || bwa.toLon != na.lons[0] {
		t.Error("reverse toLat/toLon should reflect its own SetTo")
	}
}

func TestBeelineWeightApproximator_String(t *testing.T) {
	na := &mockNodeAccess{
		lats: []float64{48.0},
		lons: []float64{11.0},
	}
	w := &beelineMockWeighting{minWeightPerDist: 0.001}
	approx := NewBeelineWeightApproximator(na, w)

	if approx.String() != "beeline" {
		t.Errorf("expected 'beeline', got %q", approx.String())
	}
}

func TestBeelineWeightApproximator_IsAdmissible(t *testing.T) {
	// The beeline approximation must not overestimate the true path weight.
	// For a straight-line path, the approximation should equal the actual weight.
	// For any realistic path, the true weight >= beeline weight.
	na := &mockNodeAccess{
		lats: []float64{48.0, 48.5, 49.0},
		lons: []float64{11.0, 11.5, 12.0},
	}
	// speed = 1000 m/s => minWeightPerDistance = 0.001 s/m
	w := &beelineMockWeighting{minWeightPerDist: 0.001}
	approx := NewBeelineWeightApproximator(na, w)
	approx.SetTo(2)

	// Direct beeline to goal from node 0
	directApprox := approx.Approximate(0)

	// Simulated true path weight through intermediate node 1
	dist01 := util.DistEarth.CalcDist(48.0, 11.0, 48.5, 11.5)
	dist12 := util.DistEarth.CalcDist(48.5, 11.5, 49.0, 12.0)
	trueWeight := (dist01 + dist12) * 0.001

	if directApprox > trueWeight {
		t.Errorf("approximation %f should not exceed true weight %f (admissibility violated)",
			directApprox, trueWeight)
	}
}
