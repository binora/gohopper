package routing

import (
	"testing"

	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// --- mocks ---

// mockWeighting implements weighting.Weighting for testing.
type mockWeighting struct {
	hasTurnCosts bool
}

func (m *mockWeighting) CalcMinWeightPerDistance() float64                              { return 1.0 }
func (m *mockWeighting) CalcEdgeWeight(_ util.EdgeIteratorState, _ bool) float64        { return 1.0 }
func (m *mockWeighting) CalcEdgeMillis(_ util.EdgeIteratorState, _ bool) int64          { return 1000 }
func (m *mockWeighting) CalcTurnWeight(_, _, _ int) float64                             { return 0 }
func (m *mockWeighting) CalcTurnMillis(_, _, _ int) int64                               { return 0 }
func (m *mockWeighting) HasTurnCosts() bool                                             { return m.hasTurnCosts }
func (m *mockWeighting) GetName() string                                                { return "mock" }

var _ weighting.Weighting = (*mockWeighting)(nil)

// mockEdgeExplorer implements util.EdgeExplorer for testing.
type mockEdgeExplorer struct{}

func (m *mockEdgeExplorer) SetBaseNode(_ int) util.EdgeIterator { return nil }

var _ util.EdgeExplorer = (*mockEdgeExplorer)(nil)

// mockNodeAccess implements storage.NodeAccess for testing.
type mockNodeAccess struct{}

func (m *mockNodeAccess) SetNode(_ int, _, _, _ float64)   {}
func (m *mockNodeAccess) GetLat(_ int) float64             { return 0 }
func (m *mockNodeAccess) GetLon(_ int) float64             { return 0 }
func (m *mockNodeAccess) GetEle(_ int) float64             { return 0 }
func (m *mockNodeAccess) Is3D() bool                       { return false }
func (m *mockNodeAccess) Dimension() int                   { return 2 }
func (m *mockNodeAccess) EnsureNode(_ int)                 {}
func (m *mockNodeAccess) GetTurnCostIndex(_ int) int       { return 0 }
func (m *mockNodeAccess) SetTurnCostIndex(_ int, _ int)    {}

var _ storage.NodeAccess = (*mockNodeAccess)(nil)

// mockGraph implements storage.Graph for testing.
type mockGraph struct {
	nodeAccess storage.NodeAccess
}

func (m *mockGraph) GetBaseGraph() *storage.BaseGraph                                       { return nil }
func (m *mockGraph) GetNodes() int                                                          { return 0 }
func (m *mockGraph) GetEdges() int                                                          { return 0 }
func (m *mockGraph) GetNodeAccess() storage.NodeAccess                                      { return m.nodeAccess }
func (m *mockGraph) GetBounds() util.BBox                                                   { return util.BBox{} }
func (m *mockGraph) Edge(_, _ int) util.EdgeIteratorState                                   { return nil }
func (m *mockGraph) GetEdgeIteratorState(_, _ int) util.EdgeIteratorState                   { return nil }
func (m *mockGraph) GetEdgeIteratorStateForKey(_ int) util.EdgeIteratorState                { return nil }
func (m *mockGraph) GetOtherNode(_, _ int) int                                              { return 0 }
func (m *mockGraph) IsAdjacentToNode(_, _ int) bool                                         { return false }
func (m *mockGraph) GetAllEdges() storage.AllEdgesIterator                                   { return nil }
func (m *mockGraph) CreateEdgeExplorer(_ routingutil.EdgeFilter) util.EdgeExplorer           { return &mockEdgeExplorer{} }
func (m *mockGraph) GetTurnCostStorage() *storage.TurnCostStorage                            { return nil }

var _ storage.Graph = (*mockGraph)(nil)

// mockEdgeIteratorState provides a minimal EdgeIteratorState for Accept tests.
type mockEdgeIteratorState struct {
	util.EdgeIteratorState
	edge int
}

func (m *mockEdgeIteratorState) GetEdge() int { return m.edge }

// --- tests ---

func TestAbstractRoutingAlgorithm_TurnCostGuard(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when using turn-cost weighting with node-based traversal, but did not panic")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected panic string, got %T: %v", r, r)
		}
		expected := "Weightings supporting turn costs cannot be used with node-based traversal mode"
		if msg != expected {
			t.Errorf("panic message = %q, want %q", msg, expected)
		}
	}()

	graph := &mockGraph{nodeAccess: &mockNodeAccess{}}
	w := &mockWeighting{hasTurnCosts: true}
	NewAbstractRoutingAlgorithm(graph, w, routingutil.NodeBased)
}

func TestAbstractRoutingAlgorithm_Accept(t *testing.T) {
	graph := &mockGraph{nodeAccess: &mockNodeAccess{}}
	w := &mockWeighting{hasTurnCosts: false}

	// Node-based mode rejects u-turns (same edge ID).
	algo := NewAbstractRoutingAlgorithm(graph, w, routingutil.NodeBased)
	edge := &mockEdgeIteratorState{edge: 5}

	if algo.Accept(edge, 5) {
		t.Error("node-based Accept should reject u-turn (same edge ID)")
	}
	if !algo.Accept(edge, 3) {
		t.Error("node-based Accept should allow different edge IDs")
	}

	// Edge-based mode always accepts (u-turn handling deferred to calcTurnWeight).
	algoEdge := NewAbstractRoutingAlgorithm(graph, w, routingutil.EdgeBased)
	if !algoEdge.Accept(edge, 5) {
		t.Error("edge-based Accept should always accept, even for same edge ID")
	}
}
