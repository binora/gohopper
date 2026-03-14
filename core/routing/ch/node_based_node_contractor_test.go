package ch

import (
	"testing"

	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"
	webapi "gohopper/web-api"

	"github.com/stretchr/testify/assert"
)

// --- node-based contractor test helpers ---

type nbcTestShortcut struct {
	baseNode  int
	adjNode   int
	weight    float64
	fwd       bool
	bwd       bool
	skipEdge1 int
	skipEdge2 int
}

func nbcCreateNodeContractor(g *storage.BaseGraph, store *storage.CHStorage, w weighting.Weighting) NodeContractor {
	prepareGraph := NewCHPreparationGraphNodeBased(g.GetNodes(), g.GetEdges())
	chGraph := storage.NewRoutingCHGraph(g, store, w)
	BuildFromGraph(prepareGraph, g, chGraph.GetWeighting())
	nc := NewNodeBasedNodeContractor(prepareGraph, storage.NewCHStorageBuilder(store), webapi.NewPMap())
	nc.InitFromGraph()
	return nc
}

func nbcContractInOrder(g *storage.BaseGraph, store *storage.CHStorage, w weighting.Weighting, nodeIDs ...int) {
	nbcSetMaxLevelOnAllNodes(store)
	b := storage.NewCHStorageBuilder(store)
	nc := nbcCreateNodeContractor(g, store, w)
	for level, n := range nodeIDs {
		b.SetLevel(n, level)
		nc.ContractNode(n)
	}
	nc.FinishContraction()
}

func nbcSetMaxLevelOnAllNodes(store *storage.CHStorage) {
	storage.NewCHStorageBuilder(store).SetLevelForAllNodes(store.GetNodes())
}

func nbcCheckShortcuts(t *testing.T, store *storage.CHStorage, expected ...nbcTestShortcut) {
	t.Helper()
	expectedSet := make(map[nbcTestShortcut]bool)
	for _, e := range expected {
		expectedSet[e] = true
	}
	assert.Equal(t, len(expected), len(expectedSet), "was given duplicate shortcuts")

	givenSet := make(map[nbcTestShortcut]bool)
	for i := 0; i < store.GetShortcuts(); i++ {
		ptr := store.ToShortcutPointer(i)
		givenSet[nbcTestShortcut{
			baseNode:  store.GetNodeA(ptr),
			adjNode:   store.GetNodeB(ptr),
			weight:    store.GetWeight(ptr),
			fwd:       store.GetFwdAccess(ptr),
			bwd:       store.GetBwdAccess(ptr),
			skipEdge1: store.GetSkippedEdge1(ptr),
			skipEdge2: store.GetSkippedEdge2(ptr),
		}] = true
	}
	assert.Equal(t, expectedSet, givenSet)
}

func nbcExpectedShortcut(baseNode, adjNode int, edge1, edge2 util.EdgeIteratorState, w weighting.Weighting, fwd, bwd bool) nbcTestShortcut {
	weight1 := w.CalcEdgeWeight(edge1, false)
	weight2 := w.CalcEdgeWeight(edge2, false)
	return nbcTestShortcut{
		baseNode:  baseNode,
		adjNode:   adjNode,
		weight:    weight1 + weight2,
		fwd:       fwd,
		bwd:       bwd,
		skipEdge1: edge1.GetEdge(),
		skipEdge2: edge2.GetEdge(),
	}
}

// --- tests ---

func TestNodeBasedNodeContractor_DirectedGraph(t *testing.T) {
	for _, reverse := range []bool{true, false} {
		name := "forward"
		if reverse {
			name = "reverse"
		}
		t.Run(name, func(t *testing.T) {
			speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
			em := routingutil.Start().Add(speedEnc).Build()
			graph := storage.NewBaseGraphBuilder(em.BytesForFlags).CreateGraph()
			w := weighting.NewSpeedWeighting(speedEnc)

			//5 6 7
			// \|/
			//4-3_1<-\ 10
			//     \_|/
			//   0___2_11
			graph.Edge(0, 2).SetDistance(200).SetDecimalBothDir(speedEnc, 10, 10)
			graph.Edge(10, 2).SetDistance(200).SetDecimalBothDir(speedEnc, 10, 10)
			graph.Edge(11, 2).SetDistance(200).SetDecimalBothDir(speedEnc, 10, 10)
			edge2to1bidirected := graph.Edge(2, 1).SetDistance(200).SetDecimalBothDir(speedEnc, 10, 10)
			graph.Edge(2, 1).SetDistance(10000).SetDecimalBothDir(speedEnc, 10, 0)
			edge1to3 := graph.Edge(1, 3).SetDistance(200).SetDecimalBothDir(speedEnc, 10, 10)
			graph.Edge(3, 4).SetDistance(200).SetDecimalBothDir(speedEnc, 10, 10)
			graph.Edge(3, 5).SetDistance(200).SetDecimalBothDir(speedEnc, 10, 10)
			graph.Edge(3, 6).SetDistance(200).SetDecimalBothDir(speedEnc, 10, 10)
			graph.Edge(3, 7).SetDistance(200).SetDecimalBothDir(speedEnc, 10, 10)
			graph.Freeze()
			store := storage.CHStorageFromGraph(graph, "profile", false)

			if reverse {
				nbcContractInOrder(graph, store, w, 1, 0, 11, 10, 4, 5, 6, 7, 3, 2)
				nbcCheckShortcuts(t, store,
					nbcExpectedShortcut(3, 2, edge1to3, edge2to1bidirected, w, true, true))
			} else {
				nbcContractInOrder(graph, store, w, 1, 0, 11, 10, 4, 5, 6, 7, 2, 3)
				nbcCheckShortcuts(t, store,
					nbcExpectedShortcut(2, 3, edge2to1bidirected, edge1to3, w, true, true))
			}
		})
	}
}

func TestNodeBasedNodeContractor_FindShortcuts_Roundabout(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	em := routingutil.Start().Add(speedEnc).Build()
	graph := storage.NewBaseGraphBuilder(em.BytesForFlags).CreateGraph()
	w := weighting.NewSpeedWeighting(speedEnc)

	// 1 -- 3 -- 4 ---> 5 ---> 6 -- 7
	//            \           /
	//             <--- 8 <---
	iter1to3 := graph.Edge(1, 3).SetDistance(100).SetDecimalBothDir(speedEnc, 10, 10)
	iter3to4 := graph.Edge(3, 4).SetDistance(100).SetDecimalBothDir(speedEnc, 10, 10)
	iter4to5 := graph.Edge(4, 5).SetDistance(100).SetDecimalBothDir(speedEnc, 10, 0)
	iter5to6 := graph.Edge(5, 6).SetDistance(100).SetDecimalBothDir(speedEnc, 10, 0)
	iter6to8 := graph.Edge(6, 8).SetDistance(200).SetDecimalBothDir(speedEnc, 10, 0)
	iter8to4 := graph.Edge(8, 4).SetDistance(100).SetDecimalBothDir(speedEnc, 10, 0)
	graph.Edge(6, 7).SetDistance(100).SetDecimalBothDir(speedEnc, 10, 10)
	graph.Freeze()
	store := storage.CHStorageFromGraph(graph, "profile", false)

	nbcContractInOrder(graph, store, w, 3, 5, 7, 8, 4, 1, 6)

	lg := storage.NewRoutingCHGraph(graph, store, w)
	nbcCheckShortcuts(t, store,
		nbcExpectedShortcut(4, 1, iter3to4, iter1to3, w, true, true),
		nbcExpectedShortcut(4, 6, iter8to4, iter6to8, w, false, true),
		nbcExpectedShortcut(4, 6, iter4to5, iter5to6, w, true, false),
		nbcExpectedShortcutFromCH(1, 6, lg, 8, 4, 7, 6, true, false),
		nbcExpectedShortcutFromCH(1, 6, lg, 8, 1, 9, 4, false, true),
	)
}

// nbcExpectedShortcutFromCH builds an expected shortcut using RoutingCHGraph edge states.
func nbcExpectedShortcutFromCH(baseNode, adjNode int, lg storage.RoutingCHGraph, edge1ID, adj1, edge2ID, adj2 int, fwd, bwd bool) nbcTestShortcut {
	e1 := lg.GetEdgeIteratorState(edge1ID, adj1)
	e2 := lg.GetEdgeIteratorState(edge2ID, adj2)
	return nbcTestShortcut{
		baseNode:  baseNode,
		adjNode:   adjNode,
		weight:    e1.GetWeight(false) + e2.GetWeight(false),
		fwd:       fwd,
		bwd:       bwd,
		skipEdge1: e1.GetEdge(),
		skipEdge2: e2.GetEdge(),
	}
}

func TestNodeBasedNodeContractor_ShortcutMergeBug(t *testing.T) {
	for _, reverse := range []bool{true, false} {
		name := "forward"
		if reverse {
			name = "reverse"
		}
		t.Run(name, func(t *testing.T) {
			speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
			em := routingutil.Start().Add(speedEnc).Build()
			graph := storage.NewBaseGraphBuilder(em.BytesForFlags).CreateGraph()
			w := weighting.NewSpeedWeighting(speedEnc)

			// ---1---->----2-----3
			//    \--------/
			edge1to2bidirected := graph.Edge(1, 2).SetDistance(200).SetDecimalBothDir(speedEnc, 10, 10)
			edge1to2directed := graph.Edge(1, 2).SetDistance(100).SetDecimalBothDir(speedEnc, 10, 0)
			edge2to3 := graph.Edge(2, 3).SetDistance(100).SetDecimalBothDir(speedEnc, 10, 10)
			graph.Freeze()
			store := storage.CHStorageFromGraph(graph, "profile", false)
			nbcSetMaxLevelOnAllNodes(store)

			if reverse {
				nbcContractInOrder(graph, store, w, 2, 1, 3)
				nbcCheckShortcuts(t, store,
					nbcExpectedShortcut(1, 3, edge1to2directed, edge2to3, w, true, false),
					nbcExpectedShortcut(1, 3, edge1to2bidirected, edge2to3, w, false, true),
				)
			} else {
				nbcContractInOrder(graph, store, w, 2, 3, 1)
				nbcCheckShortcuts(t, store,
					nbcExpectedShortcut(3, 1, edge2to3, edge1to2bidirected, w, true, false),
					nbcExpectedShortcut(3, 1, edge2to3, edge1to2directed, w, false, true),
				)
			}
		})
	}
}

func TestNodeBasedNodeContractor_ContractNode_DirectedShortcutRequired(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	em := routingutil.Start().Add(speedEnc).Build()
	graph := storage.NewBaseGraphBuilder(em.BytesForFlags).CreateGraph()
	w := weighting.NewSpeedWeighting(speedEnc)

	// 0 --> 1 --> 2
	edge1 := graph.Edge(0, 1).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	edge2 := graph.Edge(1, 2).SetDistance(2).SetDecimalBothDir(speedEnc, 60, 0)
	graph.Freeze()
	store := storage.CHStorageFromGraph(graph, "profile", false)
	nbcSetMaxLevelOnAllNodes(store)
	nbcContractInOrder(graph, store, w, 1, 0, 2)
	nbcCheckShortcuts(t, store, nbcExpectedShortcut(0, 2, edge1, edge2, w, true, false))
}

func TestNodeBasedNodeContractor_ContractNode_DirectedShortcutRequired_Reverse(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	em := routingutil.Start().Add(speedEnc).Build()
	graph := storage.NewBaseGraphBuilder(em.BytesForFlags).CreateGraph()
	w := weighting.NewSpeedWeighting(speedEnc)

	// 0 <-- 1 <-- 2
	edge1 := graph.Edge(2, 1).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	edge2 := graph.Edge(1, 0).SetDistance(2).SetDecimalBothDir(speedEnc, 60, 0)
	graph.Freeze()
	store := storage.CHStorageFromGraph(graph, "profile", false)
	nbcSetMaxLevelOnAllNodes(store)
	nbcContractInOrder(graph, store, w, 1, 2, 0)
	nbcCheckShortcuts(t, store, nbcExpectedShortcut(2, 0, edge1, edge2, w, true, false))
}

func TestNodeBasedNodeContractor_ContractNode_BidirectedShortcutsRequired(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	em := routingutil.Start().Add(speedEnc).Build()
	graph := storage.NewBaseGraphBuilder(em.BytesForFlags).CreateGraph()
	w := weighting.NewSpeedWeighting(speedEnc)

	// 0 -- 1 -- 2
	edge1 := graph.Edge(0, 1).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	edge2 := graph.Edge(1, 2).SetDistance(2).SetDecimalBothDir(speedEnc, 60, 60)
	graph.Freeze()
	store := storage.CHStorageFromGraph(graph, "profile", false)
	nbcContractInOrder(graph, store, w, 1, 2, 0)
	nbcCheckShortcuts(t, store, nbcExpectedShortcut(2, 0, edge2, edge1, w, true, true))
}

func TestNodeBasedNodeContractor_ContractNode_DirectedWithWitness(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	em := routingutil.Start().Add(speedEnc).Build()
	graph := storage.NewBaseGraphBuilder(em.BytesForFlags).CreateGraph()
	w := weighting.NewSpeedWeighting(speedEnc)

	// 0 --> 1 --> 2
	//  \_________/
	graph.Edge(0, 1).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	graph.Edge(1, 2).SetDistance(2).SetDecimalBothDir(speedEnc, 60, 0)
	graph.Edge(0, 2).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	graph.Freeze()
	store := storage.CHStorageFromGraph(graph, "profile", false)
	nbcSetMaxLevelOnAllNodes(store)
	nc := nbcCreateNodeContractor(graph, store, w)
	nc.ContractNode(1)
	// no shortcuts needed — witness path 0->2 exists
	nbcCheckShortcuts(t, store)
}
