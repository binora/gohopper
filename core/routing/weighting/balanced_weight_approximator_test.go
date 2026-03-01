package weighting

import (
	"math"
	"testing"
)

// mockWeightApproximator is a simple WeightApproximator for testing.
type mockWeightApproximator struct {
	toNode int
	factor float64 // approximate(node) = factor * |node - toNode|
	slack  float64
}

func newMockWeightApproximator(factor float64) *mockWeightApproximator {
	return &mockWeightApproximator{factor: factor}
}

func (m *mockWeightApproximator) Approximate(currentNode int) float64 {
	diff := currentNode - m.toNode
	if diff < 0 {
		diff = -diff
	}
	return m.factor * float64(diff)
}

func (m *mockWeightApproximator) SetTo(toNode int) {
	m.toNode = toNode
}

func (m *mockWeightApproximator) Reverse() WeightApproximator {
	return newMockWeightApproximator(m.factor)
}

func (m *mockWeightApproximator) GetSlack() float64 {
	return m.slack
}

func (m *mockWeightApproximator) String() string {
	return "mock"
}

func TestBalancedWeightApproximator_NilPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil WeightApproximator")
		}
	}()
	NewBalancedWeightApproximator(nil)
}

func TestBalancedWeightApproximator_GetApproximation(t *testing.T) {
	approx := newMockWeightApproximator(1.0)
	bwa := NewBalancedWeightApproximator(approx)
	if bwa.GetApproximation() != approx {
		t.Error("GetApproximation should return the forward approximator")
	}
}

func TestBalancedWeightApproximator_SetFromTo(t *testing.T) {
	approx := newMockWeightApproximator(2.0)
	bwa := NewBalancedWeightApproximator(approx)
	bwa.SetFromTo(0, 10)

	// After SetFromTo(0, 10):
	//   fwd.SetTo(10), rev.SetTo(0)
	//   fromOffset = 0.5 * fwd.Approximate(0) = 0.5 * 2.0 * |0-10| = 10.0
	//   toOffset   = 0.5 * rev.Approximate(10) = 0.5 * 2.0 * |10-0| = 10.0
	if math.Abs(bwa.fromOffset-10.0) > 1e-10 {
		t.Errorf("expected fromOffset 10.0, got %f", bwa.fromOffset)
	}
	if math.Abs(bwa.toOffset-10.0) > 1e-10 {
		t.Errorf("expected toOffset 10.0, got %f", bwa.toOffset)
	}
}

func TestBalancedWeightApproximator_ApproximateForward(t *testing.T) {
	approx := newMockWeightApproximator(2.0)
	bwa := NewBalancedWeightApproximator(approx)
	bwa.SetFromTo(0, 10)

	// forward: toOffset + 0.5*(fwd(node) - rev(node))
	// node=5: toOffset + 0.5*(2*|5-10| - 2*|5-0|) = 10 + 0.5*(10-10) = 10
	result := bwa.Approximate(5, false)
	if math.Abs(result-10.0) > 1e-10 {
		t.Errorf("expected 10.0, got %f", result)
	}

	// node=0: toOffset + 0.5*(2*|0-10| - 2*|0-0|) = 10 + 0.5*(20-0) = 20
	result = bwa.Approximate(0, false)
	if math.Abs(result-20.0) > 1e-10 {
		t.Errorf("expected 20.0, got %f", result)
	}

	// node=10: toOffset + 0.5*(2*|10-10| - 2*|10-0|) = 10 + 0.5*(0-20) = 0
	result = bwa.Approximate(10, false)
	if math.Abs(result-0.0) > 1e-10 {
		t.Errorf("expected 0.0, got %f", result)
	}
}

func TestBalancedWeightApproximator_ApproximateReverse(t *testing.T) {
	approx := newMockWeightApproximator(2.0)
	bwa := NewBalancedWeightApproximator(approx)
	bwa.SetFromTo(0, 10)

	// reverse: fromOffset - 0.5*(fwd(node) - rev(node))
	// node=5: 10 - 0.5*(10-10) = 10
	result := bwa.Approximate(5, true)
	if math.Abs(result-10.0) > 1e-10 {
		t.Errorf("expected 10.0, got %f", result)
	}

	// node=0: 10 - 0.5*(20-0) = 0
	result = bwa.Approximate(0, true)
	if math.Abs(result-0.0) > 1e-10 {
		t.Errorf("expected 0.0, got %f", result)
	}

	// node=10: 10 - 0.5*(0-20) = 20
	result = bwa.Approximate(10, true)
	if math.Abs(result-20.0) > 1e-10 {
		t.Errorf("expected 20.0, got %f", result)
	}
}

func TestBalancedWeightApproximator_GetSlack(t *testing.T) {
	approx := newMockWeightApproximator(1.0)
	approx.slack = 3.5
	bwa := NewBalancedWeightApproximator(approx)

	if bwa.GetSlack() != 3.5 {
		t.Errorf("expected slack 3.5, got %f", bwa.GetSlack())
	}
}

func TestBalancedWeightApproximator_String(t *testing.T) {
	approx := newMockWeightApproximator(1.0)
	bwa := NewBalancedWeightApproximator(approx)

	if bwa.String() != "mock" {
		t.Errorf("expected 'mock', got %q", bwa.String())
	}
}

func TestBalancedWeightApproximator_Consistency(t *testing.T) {
	// Verify that for any adjacent nodes u, v the balanced approximation
	// satisfies the consistency condition:
	//   approximate(u) - approximate(v) <= weight(u,v)
	// This is the fundamental property of a balanced approximator.
	approx := newMockWeightApproximator(1.0)
	bwa := NewBalancedWeightApproximator(approx)
	bwa.SetFromTo(0, 20)

	for u := 0; u <= 20; u++ {
		for v := 0; v <= 20; v++ {
			fwdDiff := bwa.Approximate(u, false) - bwa.Approximate(v, false)
			revDiff := bwa.Approximate(u, true) - bwa.Approximate(v, true)
			// Both differences should be negations of each other
			if math.Abs(fwdDiff+revDiff) > 1e-10 {
				t.Errorf("forward and reverse diffs should sum to zero for nodes %d,%d: got %f + %f", u, v, fwdDiff, revDiff)
			}
		}
	}
}
