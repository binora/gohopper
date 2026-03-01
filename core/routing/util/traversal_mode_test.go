package util

import (
	"testing"

	ghutil "gohopper/core/util"
)

// mockEdgeIteratorState is a minimal mock implementing only the methods
// needed by TraversalMode.CreateTraversalID.
type mockEdgeIteratorState struct {
	ghutil.EdgeIteratorState
	adjNode       int
	edgeKey       int
	reverseEdgeKey int
}

func (m *mockEdgeIteratorState) GetAdjNode() int       { return m.adjNode }
func (m *mockEdgeIteratorState) GetEdgeKey() int        { return m.edgeKey }
func (m *mockEdgeIteratorState) GetReverseEdgeKey() int { return m.reverseEdgeKey }

func TestTraversalMode_NodeBased_CreateTraversalID(t *testing.T) {
	edge := &mockEdgeIteratorState{adjNode: 42, edgeKey: 10, reverseEdgeKey: 11}

	// NodeBased always returns the adjacent node, regardless of reverse flag.
	if got := NodeBased.CreateTraversalID(edge, false); got != 42 {
		t.Errorf("NodeBased.CreateTraversalID(_, false) = %d, want 42", got)
	}
	if got := NodeBased.CreateTraversalID(edge, true); got != 42 {
		t.Errorf("NodeBased.CreateTraversalID(_, true) = %d, want 42", got)
	}
}

func TestTraversalMode_EdgeBased_CreateTraversalID(t *testing.T) {
	edge := &mockEdgeIteratorState{adjNode: 42, edgeKey: 10, reverseEdgeKey: 11}

	// EdgeBased returns edge key when forward, reverse edge key when reverse.
	if got := EdgeBased.CreateTraversalID(edge, false); got != 10 {
		t.Errorf("EdgeBased.CreateTraversalID(_, false) = %d, want 10", got)
	}
	if got := EdgeBased.CreateTraversalID(edge, true); got != 11 {
		t.Errorf("EdgeBased.CreateTraversalID(_, true) = %d, want 11", got)
	}
}

func TestTraversalMode_IsEdgeBased(t *testing.T) {
	if NodeBased.IsEdgeBased() {
		t.Error("NodeBased.IsEdgeBased() = true, want false")
	}
	if !EdgeBased.IsEdgeBased() {
		t.Error("EdgeBased.IsEdgeBased() = false, want true")
	}
}
