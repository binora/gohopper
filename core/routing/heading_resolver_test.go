package routing

import (
	"reflect"
	"testing"

	"gohopper/core/routing/ev"
	"gohopper/core/storage"
	"gohopper/core/util"
)

func TestHeadingResolver_StraightEdges(t *testing.T) {
	//    0 1 2
	//     \|/
	// 7 -- 8 --- 3
	//     /|\
	//    6 5 4
	accessEnc := ev.NewSimpleBooleanEncodedValueDir("access", true)
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, false)
	cfg := ev.NewInitializerConfig()
	accessEnc.Init(cfg)
	speedEnc.Init(cfg)
	graph := storage.NewBaseGraphBuilder(cfg.GetRequiredBytes()).CreateGraph()
	t.Cleanup(func() { graph.Close() })
	na := graph.GetNodeAccess()

	na.SetNode(0, 49.5073, 1.5545, 0)
	na.SetNode(1, 49.5002, 2.3895, 0)
	na.SetNode(2, 49.4931, 3.3013, 0)
	na.SetNode(3, 48.8574, 3.2025, 0)
	na.SetNode(4, 48.2575, 3.0651, 0)
	na.SetNode(5, 48.2393, 2.2576, 0)
	na.SetNode(6, 48.2246, 1.2249, 0)
	na.SetNode(7, 48.8611, 1.2194, 0)
	na.SetNode(8, 48.8538, 2.3950, 0)

	util.SetSpeed(60, true, true, accessEnc, speedEnc, graph.Edge(8, 0).SetDistance(10)) // edge 0
	util.SetSpeed(60, true, true, accessEnc, speedEnc, graph.Edge(8, 1).SetDistance(10)) // edge 1
	util.SetSpeed(60, true, true, accessEnc, speedEnc, graph.Edge(8, 2).SetDistance(10)) // edge 2
	util.SetSpeed(60, true, true, accessEnc, speedEnc, graph.Edge(8, 3).SetDistance(10)) // edge 3
	util.SetSpeed(60, true, true, accessEnc, speedEnc, graph.Edge(8, 4).SetDistance(10)) // edge 4
	util.SetSpeed(60, true, true, accessEnc, speedEnc, graph.Edge(8, 5).SetDistance(10)) // edge 5
	util.SetSpeed(60, true, true, accessEnc, speedEnc, graph.Edge(8, 6).SetDistance(10)) // edge 6
	util.SetSpeed(60, true, true, accessEnc, speedEnc, graph.Edge(8, 7).SetDistance(10)) // edge 7

	resolver := NewHeadingResolver(graph)

	// using default tolerance
	assertEdges(t, []int{7, 6, 0}, resolver.GetEdgesWithDifferentHeading(8, 90))
	assertEdges(t, []int{7, 6, 0}, resolver.SetTolerance(100).GetEdgesWithDifferentHeading(8, 90))
	assertEdges(t, []int{7, 6, 5, 4, 2, 1, 0}, resolver.SetTolerance(10).GetEdgesWithDifferentHeading(8, 90))
	assertEdges(t, []int{7, 6, 5, 1, 0}, resolver.SetTolerance(60).GetEdgesWithDifferentHeading(8, 90))

	assertEdges(t, []int{1}, resolver.SetTolerance(170).GetEdgesWithDifferentHeading(8, 180))
	assertEdges(t, []int{2, 1, 0}, resolver.SetTolerance(130).GetEdgesWithDifferentHeading(8, 180))

	assertEdges(t, []int{5, 4, 3}, resolver.SetTolerance(90).GetEdgesWithDifferentHeading(8, 315))
	assertEdges(t, []int{6, 5, 4, 3, 2}, resolver.SetTolerance(50).GetEdgesWithDifferentHeading(8, 315))
}

func TestHeadingResolver_CurvyEdge(t *testing.T) {
	//    1 -|
	// |- 0 -|
	// |- 2
	accessEnc := ev.NewSimpleBooleanEncodedValueDir("access", true)
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, false)
	cfg := ev.NewInitializerConfig()
	accessEnc.Init(cfg)
	speedEnc.Init(cfg)
	graph := storage.NewBaseGraphBuilder(cfg.GetRequiredBytes()).CreateGraph()
	t.Cleanup(func() { graph.Close() })
	na := graph.GetNodeAccess()

	na.SetNode(1, 0.01, 0.00, 0)
	na.SetNode(0, 0.00, 0.00, 0)
	na.SetNode(2, -0.01, 0.00, 0)

	edge0 := util.SetSpeed(60, true, true, accessEnc, speedEnc, graph.Edge(0, 1).SetDistance(10))
	edge0.SetWayGeometry(util.CreatePointList(0.00, 0.01, 0.01, 0.01))
	edge1 := util.SetSpeed(60, true, true, accessEnc, speedEnc, graph.Edge(0, 2).SetDistance(10))
	edge1.SetWayGeometry(util.CreatePointList(0.00, -0.01, -0.01, -0.01))

	resolver := NewHeadingResolver(graph)
	resolver.SetTolerance(120)

	// asking for the edges not going east returns 0-2
	assertEdges(t, []int{1}, resolver.GetEdgesWithDifferentHeading(0, 90))
	// asking for the edges not going west returns 0-1
	assertEdges(t, []int{0}, resolver.GetEdgesWithDifferentHeading(0, 270))
}

func assertEdges(t *testing.T, expected, actual []int) {
	t.Helper()
	if expected == nil {
		expected = []int{}
	}
	if actual == nil {
		actual = []int{}
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected edges %v, got %v", expected, actual)
	}
}
