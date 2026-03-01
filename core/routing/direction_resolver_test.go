package routing

import (
	"testing"

	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/storage"
	"gohopper/core/util"
)

type directionResolverFixture struct {
	accessEnc ev.BooleanEncodedValue
	speedEnc  ev.DecimalEncodedValue
	graph     *storage.BaseGraph
	na        storage.NodeAccess
}

func newDirectionResolverFixture(t *testing.T) *directionResolverFixture {
	t.Helper()
	accessEnc := ev.NewSimpleBooleanEncodedValueDir("access", true)
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, false)
	cfg := ev.NewInitializerConfig()
	accessEnc.Init(cfg)
	speedEnc.Init(cfg)
	graph := storage.NewBaseGraphBuilder(cfg.GetRequiredBytes()).CreateGraph()
	t.Cleanup(func() { graph.Close() })
	return &directionResolverFixture{
		accessEnc: accessEnc,
		speedEnc:  speedEnc,
		graph:     graph,
		na:        graph.GetNodeAccess(),
	}
}

func (f *directionResolverFixture) addNode(nodeID int, lat, lon float64) {
	f.na.SetNode(nodeID, lat, lon, 0)
}

func (f *directionResolverFixture) addEdge(from, to int, bothDirections bool) util.EdgeIteratorState {
	return util.SetSpeed(60, true, bothDirections, f.accessEnc, f.speedEnc,
		f.graph.Edge(from, to).SetDistance(1))
}

func (f *directionResolverFixture) isAccessible(edge util.EdgeIteratorState, reverse bool) bool {
	if reverse {
		return edge.GetReverseBool(f.accessEnc)
	}
	return edge.GetBool(f.accessEnc)
}

func (f *directionResolverFixture) checkResult(t *testing.T, node int, expected DirectionResolverResult) {
	t.Helper()
	f.checkResultAt(t, node, f.na.GetLat(node), f.na.GetLon(node), expected)
}

func (f *directionResolverFixture) checkResultAt(t *testing.T, node int, lat, lon float64, expected DirectionResolverResult) {
	t.Helper()
	resolver := NewDirectionResolver(f.graph, f.isAccessible)
	result := resolver.ResolveDirections(node, util.GHPoint{Lat: lat, Lon: lon})
	if result != expected {
		t.Fatalf("node %d: expected %v, got %v", node, expected, result)
	}
}

func (f *directionResolverFixture) edge(from, to int) int {
	outFilter := routingutil.EdgeFilter(func(e util.EdgeIteratorState) bool {
		return e.GetBool(f.accessEnc)
	})
	explorer := f.graph.CreateEdgeExplorer(outFilter)
	iter := explorer.SetBaseNode(from)
	for iter.Next() {
		if iter.GetAdjNode() == to {
			return iter.GetEdge()
		}
	}
	panic("could not find edge from/to")
}

func TestDirectionResolver_IsolatedNodes(t *testing.T) {
	f := newDirectionResolverFixture(t)
	f.addNode(0, 0, 0)
	f.addNode(1, 0.1, 0.1)

	f.checkResult(t, 0, Impossible())
	f.checkResult(t, 1, Impossible())
}

func TestDirectionResolver_IsolatedNodesBlockedEdge(t *testing.T) {
	f := newDirectionResolverFixture(t)
	f.addNode(0, 0, 0)
	f.addNode(1, 0.1, 0.1)
	f.graph.Edge(0, 1).SetBoolBothDir(f.accessEnc, false, false)

	f.checkResult(t, 0, Impossible())
	f.checkResult(t, 1, Impossible())
}

func TestDirectionResolver_NodesAtEndOfDeadEndStreet(t *testing.T) {
	f := newDirectionResolverFixture(t)
	//       4
	//       |
	// 0 --> 1 --> 2
	//       |
	//       3
	f.addNode(0, 2, 1.9)
	f.addNode(1, 2, 2.0)
	f.addNode(2, 2, 2.1)
	f.addNode(3, 1.9, 2.0)
	f.addNode(4, 2.1, 2.0)
	f.addEdge(0, 1, false)
	f.addEdge(1, 2, false)
	f.addEdge(1, 3, true)
	f.addEdge(1, 4, true)

	f.checkResult(t, 0, Impossible())
	f.checkResult(t, 2, Impossible())
	f.checkResult(t, 3, Restricted(f.edge(1, 3), f.edge(3, 1), f.edge(1, 3), f.edge(3, 1)))
	f.checkResult(t, 4, Restricted(f.edge(1, 4), f.edge(4, 1), f.edge(1, 4), f.edge(4, 1)))
}

func TestDirectionResolver_UnreachableNodes(t *testing.T) {
	f := newDirectionResolverFixture(t)
	//   1   3
	//  / \ /
	// 0   2
	f.addNode(0, 1, 1)
	f.addNode(1, 2, 1.5)
	f.addNode(2, 1, 2)
	f.addNode(3, 2, 2.5)
	f.addEdge(0, 1, false)
	f.addEdge(2, 1, false)
	f.addEdge(2, 3, false)

	f.checkResult(t, 1, Impossible())
	f.checkResult(t, 2, Impossible())
}

func TestDirectionResolver_Junction(t *testing.T) {
	f := newDirectionResolverFixture(t)
	//      3___
	//      |   \
	// 0 -> 1 -> 2 - 5
	//      |
	//      4
	f.addNode(0, 2.000, 1.990)
	f.addNode(1, 2.000, 2.000)
	f.addNode(2, 2.000, 2.010)
	f.addNode(3, 2.010, 2.000)
	f.addNode(4, 1.990, 2.000)
	f.addEdge(0, 1, false)
	f.addEdge(1, 2, false)
	f.addEdge(1, 3, true)
	f.addEdge(2, 3, true)
	f.addEdge(1, 4, true)
	f.addEdge(2, 5, true)

	f.checkResult(t, 1, Unrestricted())
	f.checkResult(t, 2, Unrestricted())
}

func TestDirectionResolver_JunctionExposed(t *testing.T) {
	f := newDirectionResolverFixture(t)
	// 0  1  2
	//  \ | /
	//   \|/
	//    3
	f.addNode(0, 2, 1)
	f.addNode(1, 2, 2)
	f.addNode(2, 2, 3)
	f.addNode(3, 1, 2)
	f.addEdge(0, 3, true)
	f.addEdge(1, 3, true)
	f.addEdge(2, 3, true)
	f.checkResult(t, 3, Unrestricted())
}

func TestDirectionResolver_DuplicateEdges(t *testing.T) {
	f := newDirectionResolverFixture(t)
	// 0 = 1 - 2
	f.addNode(0, 0, 0)
	f.addNode(1, 1, 1)
	f.addNode(2, 0, 2)
	f.addEdge(0, 1, true)
	f.addEdge(0, 1, true)
	f.addEdge(1, 2, true)

	f.checkResult(t, 1, Unrestricted())
	f.checkResult(t, 0, Unrestricted())
}

func TestDirectionResolver_DuplicateEdgesIn(t *testing.T) {
	f := newDirectionResolverFixture(t)
	// 0 => 1 - 2
	f.addNode(0, 1, 1)
	f.addNode(1, 2, 2)
	f.addNode(2, 1, 3)
	f.addEdge(0, 1, false)
	f.addEdge(0, 1, false)
	f.addEdge(1, 2, false)

	f.checkResult(t, 1, Unrestricted())
}

func TestDirectionResolver_DuplicateEdgesOut(t *testing.T) {
	f := newDirectionResolverFixture(t)
	// 0 - 1 => 2
	f.addNode(0, 1, 1)
	f.addNode(1, 2, 2)
	f.addNode(2, 1, 3)
	f.addEdge(0, 1, false)
	f.addEdge(1, 2, false)
	f.addEdge(1, 2, false)

	f.checkResult(t, 1, Unrestricted())
}

func TestDirectionResolver_SimpleRoad(t *testing.T) {
	f := newDirectionResolverFixture(t)
	//    x   x
	//  0-1-2-3-4
	//    x   x
	f.addNode(0, 1, 0)
	f.addNode(1, 1, 1)
	f.addNode(2, 1, 2)
	f.addNode(3, 1, 3)
	f.addNode(4, 1, 4)
	f.addNode(5, 2, 5) // ensure graph bounds are valid

	f.addEdge(0, 1, true)
	f.addEdge(1, 2, true)
	f.addEdge(2, 3, true)
	f.addEdge(3, 4, true)

	f.checkResultAt(t, 1, 1.01, 1, Restricted(f.edge(2, 1), f.edge(1, 0), f.edge(0, 1), f.edge(1, 2)))
	f.checkResultAt(t, 1, 0.99, 1, Restricted(f.edge(0, 1), f.edge(1, 2), f.edge(2, 1), f.edge(1, 0)))
	f.checkResultAt(t, 3, 1.01, 3, Restricted(f.edge(4, 3), f.edge(3, 2), f.edge(2, 3), f.edge(3, 4)))
	f.checkResultAt(t, 3, 0.99, 3, Restricted(f.edge(2, 3), f.edge(3, 4), f.edge(4, 3), f.edge(3, 2)))
}

func TestDirectionResolver_SimpleRoadOneWay(t *testing.T) {
	f := newDirectionResolverFixture(t)
	//     x     x
	//  0->1->2->3->4
	//     x     x
	f.addNode(0, 1, 0)
	f.addNode(1, 1, 1)
	f.addNode(2, 1, 2)
	f.addNode(3, 1, 3)
	f.addNode(4, 1, 4)
	f.addNode(5, 2, 5)

	f.addEdge(0, 1, false)
	f.addEdge(1, 2, false)
	f.addEdge(2, 3, false)
	f.addEdge(3, 4, false)

	f.checkResultAt(t, 1, 1.01, 1, OnlyLeft(f.edge(0, 1), f.edge(1, 2)))
	f.checkResultAt(t, 1, 0.99, 1, OnlyRight(f.edge(0, 1), f.edge(1, 2)))
	f.checkResultAt(t, 3, 1.01, 3, OnlyLeft(f.edge(2, 3), f.edge(3, 4)))
	f.checkResultAt(t, 3, 0.99, 3, OnlyRight(f.edge(2, 3), f.edge(3, 4)))
}

func TestDirectionResolver_TwoOutOneIn_OneWayRight(t *testing.T) {
	f := newDirectionResolverFixture(t)
	//     x
	// 0 - 1 -> 2
	//     x
	f.addNode(0, 0, 0)
	f.addNode(1, 1, 1)
	f.addNode(2, 2, 2)
	f.addEdge(0, 1, true)
	f.addEdge(1, 2, false)

	f.checkResultAt(t, 1, 0.99, 1, OnlyRight(0, 1))
	f.checkResultAt(t, 1, 1.01, 1, OnlyLeft(0, 1))
}

func TestDirectionResolver_TwoOutOneIn_OneWayLeft(t *testing.T) {
	f := newDirectionResolverFixture(t)
	//      x
	// 0 <- 1 - 2
	//      x
	f.addNode(0, 0, 0)
	f.addNode(1, 1, 1)
	f.addNode(2, 2, 2)
	f.addEdge(1, 0, false)
	f.addEdge(1, 2, true)

	f.checkResultAt(t, 1, 0.99, 1, OnlyLeft(1, 0))
	f.checkResultAt(t, 1, 1.01, 1, OnlyRight(1, 0))
}

func TestDirectionResolver_TwoInOneOut_OneWayRight(t *testing.T) {
	f := newDirectionResolverFixture(t)
	//     x
	// 0 - 1 <- 2
	//     x
	f.addNode(0, 0, 0)
	f.addNode(1, 1, 1)
	f.addNode(2, 2, 2)
	f.addEdge(0, 1, true)
	f.addEdge(2, 1, false)

	f.checkResultAt(t, 1, 0.99, 1, OnlyLeft(1, 0))
	f.checkResultAt(t, 1, 1.01, 1, OnlyRight(1, 0))
}

func TestDirectionResolver_TwoInOneOut_OneWayLeft(t *testing.T) {
	f := newDirectionResolverFixture(t)
	//      x
	// 0 -> 1 - 2
	//      x
	f.addNode(0, 0, 0)
	f.addNode(1, 1, 1)
	f.addNode(2, 2, 2)
	f.addEdge(0, 1, false)
	f.addEdge(2, 1, true)

	f.checkResultAt(t, 1, 0.99, 1, OnlyRight(0, 1))
	f.checkResultAt(t, 1, 1.01, 1, OnlyLeft(0, 1))
}
