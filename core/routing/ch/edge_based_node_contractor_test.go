package ch

import (
	"math"
	"testing"

	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"
	webapi "gohopper/web-api"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- edge-based contractor test helpers ---

const ebcMaxCost = 10

type ebcTestShortcut struct {
	baseNode     int
	adjNode      int
	origKeyFirst int
	origKeyLast  int
	skipEdge1    int
	skipEdge2    int
	weight       float64
	fwd          bool
	bwd          bool
}

type ebcTestFixture struct {
	t           *testing.T
	speedEnc    ev.DecimalEncodedValue
	turnCostEnc ev.DecimalEncodedValue
	graph       *storage.BaseGraph
	chStore     *storage.CHStorage
	chBuilder   *storage.CHStorageBuilder
	weighting   weighting.Weighting
	chConfigs   []*CHConfig
}

func newEBCTestFixture(t *testing.T) *ebcTestFixture {
	t.Helper()
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	turnCostEnc := ev.TurnCostCreate("car", ebcMaxCost)
	em := routingutil.Start().Add(speedEnc).AddTurnCostEncodedValue(turnCostEnc).Build()
	graph := storage.NewBaseGraphBuilder(em.BytesForFlags).SetWithTurnCosts(true).CreateGraph()
	chConfigs := []*CHConfig{
		NewCHConfigEdgeBased("p1", weighting.NewSpeedWeightingWithTurnCosts(speedEnc, turnCostEnc, graph.GetTurnCostStorage(), graph.GetNodeAccess(), math.Inf(1))),
		NewCHConfigEdgeBased("p2", weighting.NewSpeedWeightingWithTurnCosts(speedEnc, turnCostEnc, graph.GetTurnCostStorage(), graph.GetNodeAccess(), 60)),
		NewCHConfigEdgeBased("p3", weighting.NewSpeedWeightingWithTurnCosts(speedEnc, turnCostEnc, graph.GetTurnCostStorage(), graph.GetNodeAccess(), 0)),
	}
	return &ebcTestFixture{
		t:           t,
		speedEnc:    speedEnc,
		turnCostEnc: turnCostEnc,
		graph:       graph,
		chConfigs:   chConfigs,
	}
}

func (f *ebcTestFixture) freeze() {
	f.graph.Freeze()
	f.chStore = storage.CHStorageFromGraph(f.graph, f.chConfigs[0].GetName(), f.chConfigs[0].IsEdgeBased())
	f.chBuilder = storage.NewCHStorageBuilder(f.chStore)
	f.weighting = f.chConfigs[0].GetWeighting()
}

func (f *ebcTestFixture) freezeWithConfig(idx int) {
	f.graph.Freeze()
	f.chStore = storage.CHStorageFromGraph(f.graph, f.chConfigs[idx].GetName(), f.chConfigs[idx].IsEdgeBased())
	f.chBuilder = storage.NewCHStorageBuilder(f.chStore)
	f.weighting = f.chConfigs[idx].GetWeighting()
}

func (f *ebcTestFixture) setMaxLevelOnAllNodes() {
	f.chBuilder.SetLevelForAllNodes(f.chStore.GetNodes())
}

func (f *ebcTestFixture) setRestrictionByEdge(inEdge, outEdge util.EdgeIteratorState, viaNode int) {
	f.setTurnCostByEdge(inEdge, outEdge, viaNode, math.Inf(1))
}

func (f *ebcTestFixture) setRestriction(from, via, to int) {
	f.setTurnCost(from, via, to, math.Inf(1))
}

func (f *ebcTestFixture) setTurnCost(from, via, to int, cost float64) {
	edge1 := ebcGetEdge(f.graph, from, via)
	edge2 := ebcGetEdge(f.graph, via, to)
	cost1 := cost
	if cost >= ebcMaxCost {
		cost1 = math.Inf(1)
	}
	f.graph.GetTurnCostStorage().SetDecimal(f.graph.GetNodeAccess(), f.turnCostEnc, edge1.GetEdge(), via, edge2.GetEdge(), cost1)
}

func (f *ebcTestFixture) setTurnCostByEdge(inEdge, outEdge util.EdgeIteratorState, viaNode int, cost float64) {
	cost1 := cost
	if cost >= ebcMaxCost {
		cost1 = math.Inf(1)
	}
	f.graph.GetTurnCostStorage().SetDecimal(f.graph.GetNodeAccess(), f.turnCostEnc, inEdge.GetEdge(), viaNode, outEdge.GetEdge(), cost1)
}

func (f *ebcTestFixture) createNodeContractor() *EdgeBasedNodeContractor {
	turnCostFunction := BuildTurnCostFunctionFromWeighting(f.weighting)
	prepareGraph := NewCHPreparationGraphEdgeBased(f.graph.GetNodes(), f.graph.GetEdges(), turnCostFunction)
	BuildFromGraph(prepareGraph, f.graph, f.weighting)
	nc := NewEdgeBasedNodeContractor(prepareGraph, f.chBuilder, webapi.NewPMap())
	nc.InitFromGraph()
	return nc
}

func (f *ebcTestFixture) contractNode(nc NodeContractor, node, level int) {
	f.chBuilder.SetLevel(node, level)
	nc.ContractNode(node)
}

func (f *ebcTestFixture) contractNodes(nodes ...int) {
	nc := f.createNodeContractor()
	for i, n := range nodes {
		f.chBuilder.SetLevel(n, i)
		nc.ContractNode(n)
	}
	nc.FinishContraction()
}

func (f *ebcTestFixture) contractAllNodesInOrder() {
	nc := f.createNodeContractor()
	for node := 0; node < f.graph.GetNodes(); node++ {
		f.chBuilder.SetLevel(node, node)
		nc.ContractNode(node)
	}
	nc.FinishContraction()
}

func ebcGetEdge(graph *storage.BaseGraph, from, to int) util.EdgeIteratorState {
	iter := graph.CreateEdgeExplorer(routingutil.AllEdges).SetBaseNode(from)
	for iter.Next() {
		if iter.GetAdjNode() == to {
			return iter.Detach(false)
		}
	}
	panic("edge not found")
}

func ebcCheckShortcuts(t *testing.T, store *storage.CHStorage, expected ...ebcTestShortcut) {
	t.Helper()
	expectedSet := make(map[ebcTestShortcut]bool)
	for _, e := range expected {
		expectedSet[e] = true
	}
	if len(expected) != len(expectedSet) {
		t.Fatal("was given duplicate shortcuts")
	}
	givenSet := make(map[ebcTestShortcut]bool)
	for i := 0; i < store.GetShortcuts(); i++ {
		ptr := store.ToShortcutPointer(i)
		givenSet[ebcTestShortcut{
			baseNode:     store.GetNodeA(ptr),
			adjNode:      store.GetNodeB(ptr),
			origKeyFirst: store.GetOrigEdgeKeyFirst(ptr),
			origKeyLast:  store.GetOrigEdgeKeyLast(ptr),
			skipEdge1:    store.GetSkippedEdge1(ptr),
			skipEdge2:    store.GetSkippedEdge2(ptr),
			weight:       store.GetWeight(ptr),
			fwd:          store.GetFwdAccess(ptr),
			bwd:          store.GetBwdAccess(ptr),
		}] = true
	}
	assert.Equal(t, expectedSet, givenSet)
}

func ebcCheckShortcutsSet(t *testing.T, store *storage.CHStorage, expected map[ebcTestShortcut]bool) {
	t.Helper()
	givenSet := make(map[ebcTestShortcut]bool)
	for i := 0; i < store.GetShortcuts(); i++ {
		ptr := store.ToShortcutPointer(i)
		givenSet[ebcTestShortcut{
			baseNode:     store.GetNodeA(ptr),
			adjNode:      store.GetNodeB(ptr),
			origKeyFirst: store.GetOrigEdgeKeyFirst(ptr),
			origKeyLast:  store.GetOrigEdgeKeyLast(ptr),
			skipEdge1:    store.GetSkippedEdge1(ptr),
			skipEdge2:    store.GetSkippedEdge2(ptr),
			weight:       store.GetWeight(ptr),
			fwd:          store.GetFwdAccess(ptr),
			bwd:          store.GetBwdAccess(ptr),
		}] = true
	}
	assert.Equal(t, expected, givenSet)
}

func ebcCheckNumShortcuts(t *testing.T, store *storage.CHStorage, expected int) {
	t.Helper()
	assert.Equal(t, expected, store.GetShortcuts())
}

// createShortcutFromEdges creates a shortcut from two EdgeIteratorStates, computing weight from edges.
func (f *ebcTestFixture) createShortcutFromEdges(from, to int, edge1, edge2 util.EdgeIteratorState, weight float64) ebcTestShortcut {
	return f.createShortcutFromEdgesDir(from, to, edge1, edge2, weight, true, false)
}

func (f *ebcTestFixture) createShortcutFromEdgesDir(from, to int, edge1, edge2 util.EdgeIteratorState, weight float64, fwd, bwd bool) ebcTestShortcut {
	return ebcTestShortcut{
		baseNode:     from,
		adjNode:      to,
		origKeyFirst: edge1.GetEdgeKey(),
		origKeyLast:  edge2.GetEdgeKey(),
		skipEdge1:    edge1.GetEdge(),
		skipEdge2:    edge2.GetEdge(),
		weight:       weight,
		fwd:          fwd,
		bwd:          bwd,
	}
}

// createShortcutRaw creates a shortcut from raw values.
func createShortcutRaw(from, to, firstOrigEdgeKey, lastOrigEdgeKey, skipEdge1, skipEdge2 int, weight float64) ebcTestShortcut {
	return ebcTestShortcut{
		baseNode:     from,
		adjNode:      to,
		origKeyFirst: firstOrigEdgeKey,
		origKeyLast:  lastOrigEdgeKey,
		skipEdge1:    skipEdge1,
		skipEdge2:    skipEdge2,
		weight:       weight,
		fwd:          true,
		bwd:          false,
	}
}

func createShortcutRawDir(from, to, firstOrigEdgeKey, lastOrigEdgeKey, skipEdge1, skipEdge2 int, weight float64, fwd, bwd bool) ebcTestShortcut {
	return ebcTestShortcut{
		baseNode:     from,
		adjNode:      to,
		origKeyFirst: firstOrigEdgeKey,
		origKeyLast:  lastOrigEdgeKey,
		skipEdge1:    skipEdge1,
		skipEdge2:    skipEdge2,
		weight:       weight,
		fwd:          fwd,
		bwd:          bwd,
	}
}

// --- tests ---

func TestEdgeBasedContractNodes_simpleLoop(t *testing.T) {
	f := newEBCTestFixture(t)
	//     2-3
	//     | |
	//  6- 7-8
	//     |
	//     9
	f.graph.Edge(6, 7).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	edge7to8 := f.graph.Edge(7, 8).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	edge8to3 := f.graph.Edge(8, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	edge3to2 := f.graph.Edge(3, 2).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	edge2to7 := f.graph.Edge(2, 7).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(7, 9).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.freeze()
	f.setMaxLevelOnAllNodes()

	f.setRestriction(6, 7, 9)
	f.setTurnCost(8, 3, 2, 2)

	f.contractNodes(5, 6, 3, 2, 9, 1, 8, 4, 7, 0)
	ebcCheckShortcuts(t, f.chStore,
		f.createShortcutFromEdgesDir(2, 8, edge8to3, edge3to2, 5, false, true),
		createShortcutRawDir(8, 7, edge8to3.GetEdgeKey(), edge2to7.GetEdgeKey(), 6, edge2to7.GetEdge(), 6, true, false),
		createShortcutRawDir(7, 7, edge7to8.GetEdgeKey(), edge2to7.GetEdgeKey(), edge7to8.GetEdge(), 7, 8, true, false),
	)
}

func TestEdgeBasedContractNodes_necessaryAlternative(t *testing.T) {
	f := newEBCTestFixture(t)
	//      1
	//      |    can't go 1->6->3
	//      v
	// 2 -> 6 -> 3 -> 5 -> 4
	//      |    ^
	//      -> 0-|
	e6to0 := f.graph.Edge(6, 0).SetDistance(40).SetDecimalBothDir(f.speedEnc, 10, 0)
	e0to3 := f.graph.Edge(0, 3).SetDistance(50).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 6).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	e6to3 := f.graph.Edge(6, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	e3to5 := f.graph.Edge(3, 5).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 6).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(5, 4).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.setRestriction(1, 6, 3)
	f.contractAllNodesInOrder()
	ebcCheckShortcuts(t, f.chStore,
		// from contracting node 0: need a shortcut because of turn restriction
		f.createShortcutFromEdgesDir(3, 6, e6to0, e0to3, 9, false, true),
		// from contracting node 3: two shortcuts:
		// 1) in case we come from 1->6 (cant turn left)
		// 2) in case we come from 2->6 (going via node 0 would be more expensive)
		createShortcutRawDir(5, 6, e6to0.GetEdgeKey(), e3to5.GetEdgeKey(), 7, e3to5.GetEdge(), 11, false, true),
		f.createShortcutFromEdgesDir(5, 6, e6to3, e3to5, 3, false, true),
	)
}

func TestEdgeBasedContractNodes_alternativeNecessary_noUTurn(t *testing.T) {
	f := newEBCTestFixture(t)
	//    /->0-->
	//   v       \
	//  4 <-----> 2 -> 3 -> 1
	e0to4 := f.graph.Edge(4, 0).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 10)
	e0to2 := f.graph.Edge(0, 2).SetDistance(50).SetDecimalBothDir(f.speedEnc, 10, 0)
	e2to3 := f.graph.Edge(2, 3).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	_ = f.graph.Edge(3, 1).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	e2to4 := f.graph.Edge(4, 2).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.freeze()

	f.setMaxLevelOnAllNodes()
	f.contractAllNodesInOrder()
	ebcCheckShortcuts(t, f.chStore,
		// from contraction of node 0
		f.createShortcutFromEdgesDir(2, 4, e0to4, e0to2, 8, false, true),
		// from contraction of node 2
		createShortcutRawDir(3, 4, e0to4.GetEdgeKey(), e2to3.GetEdgeKey(), 5, e2to3.GetEdge(), 10, false, true),
		f.createShortcutFromEdgesDir(3, 4, e2to4, e2to3, 4, false, true),
	)
}

func TestEdgeBasedContractNodes_bidirectionalLoop(t *testing.T) {
	f := newEBCTestFixture(t)
	//  1   3
	//  |  /|
	//  0-4-6
	//    |
	//    5-2
	f.graph.Edge(1, 0).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(0, 4).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	e4to6 := f.graph.Edge(4, 6).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 10)
	e6to3 := f.graph.Edge(6, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	e3to4 := f.graph.Edge(3, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	e4to5 := f.graph.Edge(4, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(5, 2).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.freeze()

	// enforce loop (going counter-clockwise)
	f.setRestriction(0, 4, 5)
	f.setTurnCost(6, 3, 4, 2)
	f.setTurnCost(4, 3, 6, 4)
	f.setMaxLevelOnAllNodes()

	f.contractAllNodesInOrder()
	ebcCheckShortcuts(t, f.chStore,
		// from contraction of node 3
		f.createShortcutFromEdgesDir(4, 6, e3to4.Detach(true), e6to3.Detach(true), 6, true, false),
		f.createShortcutFromEdgesDir(4, 6, e6to3, e3to4, 4, false, true),
		// from contraction of node 4
		// two 'parallel' shortcuts to preserve shortest paths to 5 when coming from 4->6 and 3->6 !!
		createShortcutRawDir(5, 6, e6to3.GetEdgeKey(), e4to5.GetEdgeKey(), 8, e4to5.GetEdge(), 5, false, true),
		f.createShortcutFromEdgesDir(5, 6, e4to6.Detach(true), e4to5, 3, false, true),
	)
}

func TestEdgeBasedContractNode_twoNormalEdges_noSourceEdgeToConnect(t *testing.T) {
	f := newEBCTestFixture(t)
	// 1 --> 0 --> 2 --> 3
	f.graph.Edge(1, 0).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(0, 2).SetDistance(50).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(0, 3, 1, 2)
	ebcCheckShortcuts(t, f.chStore)
}

func TestEdgeBasedContractNode_twoNormalEdges_noTargetEdgeToConnect(t *testing.T) {
	f := newEBCTestFixture(t)
	// 3 --> 1 --> 0 --> 2
	f.graph.Edge(3, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 0).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(0, 2).SetDistance(50).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(0, 3, 1, 2)
	ebcCheckShortcuts(t, f.chStore)
}

func TestEdgeBasedContractNode_twoNormalEdges_noEdgesToConnectBecauseOfTurnRestrictions(t *testing.T) {
	f := newEBCTestFixture(t)
	// 0 --> 3 --> 2 --> 4 --> 1
	f.graph.Edge(0, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 2).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 4).SetDistance(50).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(4, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.setRestriction(0, 3, 2)
	f.setRestriction(2, 4, 1)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(2, 0, 3, 4, 1)
	ebcCheckShortcuts(t, f.chStore)
}

func TestEdgeBasedContractNode_twoNormalEdges_noTurncosts(t *testing.T) {
	f := newEBCTestFixture(t)
	// 0 --> 3 --> 2 --> 4 --> 1
	f.graph.Edge(0, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	e3to2 := f.graph.Edge(3, 2).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	e2to4 := f.graph.Edge(2, 4).SetDistance(50).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(4, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	nc := f.createNodeContractor()
	f.contractNode(nc, 0, 0)
	f.contractNode(nc, 1, 1)
	// no shortcuts so far
	ebcCheckShortcuts(t, f.chStore)
	f.contractNode(nc, 2, 2)
	f.contractNode(nc, 3, 3)
	f.contractNode(nc, 4, 4)
	nc.FinishContraction()
	ebcCheckShortcuts(t, f.chStore, f.createShortcutFromEdges(3, 4, e3to2, e2to4, 8))
}

func TestEdgeBasedContractNode_twoNormalEdges_noShortcuts(t *testing.T) {
	f := newEBCTestFixture(t)
	// 0 --> 1 --> 2 --> 3 --> 4
	f.graph.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 2).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 3).SetDistance(50).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractAllNodesInOrder()
	ebcCheckShortcuts(t, f.chStore)
}

func TestEdgeBasedContractNode_twoNormalEdges_noOutgoingEdges(t *testing.T) {
	f := newEBCTestFixture(t)
	// 0 --> 1 --> 2 <-- 3 <-- 4
	f.graph.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 2).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 2).SetDistance(50).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(4, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(2, 0, 4, 1, 3)
	ebcCheckShortcuts(t, f.chStore)
}

func TestEdgeBasedContractNode_twoNormalEdges_noIncomingEdges(t *testing.T) {
	f := newEBCTestFixture(t)
	// 0 <-- 1 <-- 2 --> 3 --> 4
	f.graph.Edge(1, 0).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 1).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 3).SetDistance(50).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(2, 0, 4, 1, 3)
	ebcCheckShortcuts(t, f.chStore)
}

func TestEdgeBasedContractNode_duplicateOutgoingEdges_differentWeight(t *testing.T) {
	f := newEBCTestFixture(t)
	// 0 -> 1 -> 2 -> 3 -> 4
	//            \->/
	f.graph.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 3).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(2, 0, 4, 1, 3)
	// there should be only one shortcut
	ebcCheckShortcuts(t, f.chStore,
		createShortcutRaw(1, 3, 2, 6, 1, 3, 2),
	)
}

func TestEdgeBasedContractNode_duplicateIncomingEdges_differentWeight(t *testing.T) {
	f := newEBCTestFixture(t)
	// 0 -> 1 -> 2 -> 3 -> 4
	//       \->/
	f.graph.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 2).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(2, 0, 4, 1, 3)
	ebcCheckShortcuts(t, f.chStore,
		createShortcutRaw(1, 3, 4, 6, 2, 3, 2),
	)
}

func TestEdgeBasedContractNode_duplicateOutgoingEdges_sameWeight(t *testing.T) {
	f := newEBCTestFixture(t)
	// 0 -> 1 -> 2 -> 3 -> 4
	//            \->/
	f.graph.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(2, 0, 4, 1, 3)
	ebcCheckNumShortcuts(t, f.chStore, 1)
}

func TestEdgeBasedContractNode_duplicateIncomingEdges_sameWeight(t *testing.T) {
	// @RepeatedTest(10) -> run 10 times
	for range 10 {
		f := newEBCTestFixture(t)
		// 0 -> 1 -> 2 -> 3 -> 4
		//       \->/
		f.graph.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
		f.graph.Edge(1, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
		f.graph.Edge(1, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
		f.graph.Edge(2, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
		f.graph.Edge(3, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
		f.freeze()
		f.setMaxLevelOnAllNodes()
		f.contractNodes(2, 0, 4, 1, 3)
		ebcCheckNumShortcuts(t, f.chStore, 1)
	}
}

func TestEdgeBasedContractNode_twoNormalEdges_withTurnCost(t *testing.T) {
	f := newEBCTestFixture(t)
	// 0 --> 3 --> 2 --> 4 --> 1
	f.graph.Edge(0, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	e3to2 := f.graph.Edge(3, 2).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	e2to4 := f.graph.Edge(2, 4).SetDistance(50).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(4, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.setTurnCost(3, 2, 4, 4)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(2, 0, 1, 3, 4)
	ebcCheckShortcuts(t, f.chStore, f.createShortcutFromEdges(3, 4, e3to2, e2to4, 12))
}

func TestEdgeBasedContractNode_twoNormalEdges_withTurnRestriction(t *testing.T) {
	f := newEBCTestFixture(t)
	// 0 --> 3 --> 2 --> 4 --> 1
	f.graph.Edge(0, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 2).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 4).SetDistance(50).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(4, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.setRestriction(3, 2, 4)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(2, 0, 1, 3, 4)
	ebcCheckShortcuts(t, f.chStore)
}

func TestEdgeBasedContractNode_twoNormalEdges_bidirectional(t *testing.T) {
	f := newEBCTestFixture(t)
	// 0 -- 3 -- 2 -- 4 -- 1
	f.graph.Edge(0, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	e3to2 := f.graph.Edge(3, 2).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 10)
	e2to4 := f.graph.Edge(2, 4).SetDistance(50).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(4, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.setTurnCostByEdge(e3to2, e2to4, 2, 4)
	f.setTurnCostByEdge(e2to4, e3to2, 2, 4)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(2, 0, 1, 3, 4)
	ebcCheckShortcuts(t, f.chStore,
		createShortcutRawDir(3, 4, 2, 4, 1, 2, 12, true, false),
		createShortcutRawDir(3, 4, 5, 3, 2, 1, 12, false, true),
	)
}

func TestEdgeBasedContractNode_twoNormalEdges_bidirectional_differentCosts(t *testing.T) {
	f := newEBCTestFixture(t)
	// 0 -- 3 -- 2 -- 4 -- 1
	f.graph.Edge(0, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	e3to2 := f.graph.Edge(3, 2).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 10)
	e2to4 := f.graph.Edge(2, 4).SetDistance(50).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(4, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.setTurnCostByEdge(e3to2, e2to4, 2, 4)
	f.setTurnCostByEdge(e2to4, e3to2, 2, 7)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(2, 0, 1, 3, 4)
	ebcCheckShortcuts(t, f.chStore,
		f.createShortcutFromEdgesDir(3, 4, e3to2, e2to4, 12, true, false),
		f.createShortcutFromEdgesDir(3, 4, e2to4.Detach(true), e3to2.Detach(true), 15, false, true),
	)
}

func TestEdgeBasedContractNode_multiple_bidirectional_linear(t *testing.T) {
	f := newEBCTestFixture(t)
	// 3 -- 2 -- 1 -- 4
	f.graph.Edge(3, 2).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(2, 1).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(1, 4).SetDistance(60).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.freeze()
	f.setMaxLevelOnAllNodes()

	f.contractNodes(1, 2, 3, 4)
	// no shortcuts needed
	ebcCheckShortcuts(t, f.chStore)
}

func TestEdgeBasedContractNode_shortcutDoesNotSpanUTurn(t *testing.T) {
	f := newEBCTestFixture(t)
	// 2 -> 7 -> 3 -> 5 -> 6
	//           |
	//     1 <-> 4
	e7to3 := f.graph.Edge(7, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	e3to5 := f.graph.Edge(3, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	e3to4 := f.graph.Edge(3, 4).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(2, 7).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(5, 6).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.setRestriction(7, 3, 5)
	f.contractNodes(3, 4, 2, 6, 7, 5, 1)
	ebcCheckShortcuts(t, f.chStore,
		// from contracting node 3
		f.createShortcutFromEdgesDir(4, 7, e7to3, e3to4, 3, false, true),
		f.createShortcutFromEdgesDir(4, 5, e3to4.Detach(true), e3to5, 3, true, false),
		// important! no shortcut from 7 to 5 when contracting node 4, because it includes a u-turn
	)
}

func TestEdgeBasedContractNode_multiple_loops_directTurnIsBest(t *testing.T) {
	f := newEBCTestFixture(t)
	g := newGraphWithTwoLoops(f, ebcMaxCost, ebcMaxCost, 1, 2, 3, 4)
	g.contractAndCheckShortcuts(t,
		f.createShortcutFromEdgesDir(7, 8, g.e7to6, g.e6to8, 11, true, false),
	)
}

func TestEdgeBasedContractNode_multiple_loops_leftLoopIsBest(t *testing.T) {
	f := newEBCTestFixture(t)
	g := newGraphWithTwoLoops(f, 2, ebcMaxCost, 1, 2, 3, ebcMaxCost)
	g.contractAndCheckShortcuts(t,
		createShortcutRawDir(6, 7, g.e7to6.GetEdgeKey(), g.e1to6.GetEdgeKey(), g.e7to6.GetEdge(), g.getScEdge(3), 12, false, true),
		createShortcutRawDir(7, 8, g.e7to6.GetEdgeKey(), g.e6to8.GetEdgeKey(), g.getScEdge(4), g.e6to8.GetEdge(), 20, true, false),
	)
}

func TestEdgeBasedContractNode_multiple_loops_rightLoopIsBest(t *testing.T) {
	f := newEBCTestFixture(t)
	g := newGraphWithTwoLoops(f, 8, 1, 1, 2, 3, ebcMaxCost)
	g.contractAndCheckShortcuts(t,
		createShortcutRawDir(6, 7, g.e7to6.GetEdgeKey(), g.e3to6.GetEdgeKey(), g.e7to6.GetEdge(), g.getScEdge(2), 12, false, true),
		createShortcutRawDir(7, 8, g.e7to6.GetEdgeKey(), g.e6to8.GetEdgeKey(), g.getScEdge(4), g.e6to8.GetEdge(), 21, true, false),
	)
}

func TestEdgeBasedContractNode_multiple_loops_leftRightLoopIsBest(t *testing.T) {
	f := newEBCTestFixture(t)
	g := newGraphWithTwoLoops(f, 3, ebcMaxCost, 1, ebcMaxCost, 3, ebcMaxCost)
	g.contractAndCheckShortcuts(t,
		createShortcutRawDir(6, 7, g.e7to6.GetEdgeKey(), g.e1to6.GetEdgeKey(), g.e7to6.GetEdge(), g.getScEdge(3), 13, false, true),
		createShortcutRawDir(6, 7, g.e7to6.GetEdgeKey(), g.e3to6.GetEdgeKey(), g.getScEdge(5), g.getScEdge(2), 24, false, true),
		createShortcutRawDir(7, 8, g.e7to6.GetEdgeKey(), g.e6to8.GetEdgeKey(), g.getScEdge(4), g.e6to8.GetEdge(), 33, true, false),
	)
}

func TestEdgeBasedContractNode_multiple_loops_rightLeftLoopIsBest(t *testing.T) {
	f := newEBCTestFixture(t)
	g := newGraphWithTwoLoops(f, ebcMaxCost, 5, 4, 2, ebcMaxCost, ebcMaxCost)
	g.contractAndCheckShortcuts(t,
		createShortcutRawDir(6, 7, g.e7to6.GetEdgeKey(), g.e3to6.GetEdgeKey(), g.e7to6.GetEdge(), g.getScEdge(2), 16, false, true),
		createShortcutRawDir(6, 7, g.e7to6.GetEdgeKey(), g.e1to6.GetEdgeKey(), g.getScEdge(5), g.getScEdge(3), 25, false, true),
		createShortcutRawDir(7, 8, g.e7to6.GetEdgeKey(), g.e6to8.GetEdgeKey(), g.getScEdge(4), g.e6to8.GetEdge(), 33, true, false),
	)
}

//    1 4 2
//    |\|/|
//    0-6-3
//     /|\
// 9--7 5 8--10
type graphWithTwoLoops struct {
	f         *ebcTestFixture
	numEdges  int
	e0to1     util.EdgeIteratorState
	e1to6     util.EdgeIteratorState
	e6to0     util.EdgeIteratorState
	e2to3     util.EdgeIteratorState
	e3to6     util.EdgeIteratorState
	e6to2     util.EdgeIteratorState
	e7to6     util.EdgeIteratorState
	e6to8     util.EdgeIteratorState
	e9to7     util.EdgeIteratorState
	e8to10    util.EdgeIteratorState
	e4to6     util.EdgeIteratorState
	e5to6     util.EdgeIteratorState
}

func newGraphWithTwoLoops(f *ebcTestFixture, turnCost70, turnCost72, turnCost12, turnCost18, turnCost38, turnCost78 int) *graphWithTwoLoops {
	centerNode := 6
	g := &graphWithTwoLoops{f: f, numEdges: 12}
	g.e0to1 = f.graph.Edge(0, 1).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e1to6 = f.graph.Edge(1, 6).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e6to0 = f.graph.Edge(6, 0).SetDistance(40).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e2to3 = f.graph.Edge(2, 3).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e3to6 = f.graph.Edge(3, 6).SetDistance(70).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e6to2 = f.graph.Edge(6, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e7to6 = f.graph.Edge(7, 6).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e6to8 = f.graph.Edge(6, 8).SetDistance(60).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e9to7 = f.graph.Edge(9, 7).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e8to10 = f.graph.Edge(8, 10).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	// these two edges help to avoid loop avoidance for the left and right loops
	g.e4to6 = f.graph.Edge(4, 6).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e5to6 = f.graph.Edge(5, 6).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)

	f.setTurnCostByEdge(g.e7to6, g.e6to0, centerNode, float64(turnCost70))
	f.setTurnCostByEdge(g.e7to6, g.e6to2, centerNode, float64(turnCost72))
	f.setTurnCostByEdge(g.e7to6, g.e6to8, centerNode, float64(turnCost78))
	f.setTurnCostByEdge(g.e1to6, g.e6to2, centerNode, float64(turnCost12))
	f.setTurnCostByEdge(g.e1to6, g.e6to8, centerNode, float64(turnCost18))
	f.setTurnCostByEdge(g.e3to6, g.e6to8, centerNode, float64(turnCost38))
	// restrictions to make sure that no loop avoidance takes place when the left&right loops are contracted
	f.setRestrictionByEdge(g.e4to6, g.e6to8, centerNode)
	f.setRestrictionByEdge(g.e5to6, g.e6to2, centerNode)
	f.setRestrictionByEdge(g.e4to6, g.e6to0, centerNode)

	f.freeze()
	f.setMaxLevelOnAllNodes()
	return g
}

func (g *graphWithTwoLoops) getScEdge(shortcutID int) int {
	return g.numEdges + shortcutID
}

func (g *graphWithTwoLoops) contractAndCheckShortcuts(t *testing.T, extraShortcuts ...ebcTestShortcut) {
	t.Helper()
	g.f.contractNodes(0, 1, 2, 3, 4, 5, 6, 9, 10, 7, 8)
	expectedSet := make(map[ebcTestShortcut]bool)
	// base shortcuts from contracting the loops
	baseShortcuts := []ebcTestShortcut{
		g.f.createShortcutFromEdgesDir(1, 6, g.e6to0, g.e0to1, 7, false, true),
		createShortcutRawDir(6, 6, g.e6to0.GetEdgeKey(), g.e1to6.GetEdgeKey(), g.getScEdge(0), g.e1to6.GetEdge(), 9, true, false),
		g.f.createShortcutFromEdgesDir(3, 6, g.e6to2, g.e2to3, 3, false, true),
		createShortcutRawDir(6, 6, g.e6to2.GetEdgeKey(), g.e3to6.GetEdgeKey(), g.getScEdge(1), g.e3to6.GetEdge(), 10, true, false),
	}
	for _, sc := range baseShortcuts {
		expectedSet[sc] = true
	}
	for _, sc := range extraShortcuts {
		expectedSet[sc] = true
	}
	ebcCheckShortcutsSet(t, g.f.chStore, expectedSet)
}

func TestEdgeBasedContractNode_detour_detourIsBetter(t *testing.T) {
	f := newEBCTestFixture(t)
	g := newGraphWithDetour(f, 2, 9, 5, 1)
	f.contractNodes(0, 4, 3, 1, 2)
	ebcCheckShortcuts(t, f.chStore,
		f.createShortcutFromEdges(1, 2, g.e1to0, g.e0to2, 7),
	)
}

func TestEdgeBasedContractNode_detour_detourIsWorse(t *testing.T) {
	f := newEBCTestFixture(t)
	_ = newGraphWithDetour(f, 4, 1, 1, 7)
	f.contractNodes(0, 4, 3, 1, 2)
	ebcCheckShortcuts(t, f.chStore)
}

//      0
//     / \
// 4--1---2--3
type graphWithDetour struct {
	e4to1 util.EdgeIteratorState
	e1to0 util.EdgeIteratorState
	e1to2 util.EdgeIteratorState
	e0to2 util.EdgeIteratorState
	e2to3 util.EdgeIteratorState
}

func newGraphWithDetour(f *ebcTestFixture, turnCost42, turnCost13, turnCost40, turnCost03 int) *graphWithDetour {
	g := &graphWithDetour{}
	g.e4to1 = f.graph.Edge(4, 1).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e1to0 = f.graph.Edge(1, 0).SetDistance(40).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e1to2 = f.graph.Edge(1, 2).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e0to2 = f.graph.Edge(0, 2).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e2to3 = f.graph.Edge(2, 3).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.setTurnCostByEdge(g.e4to1, g.e1to2, 1, float64(turnCost42))
	f.setTurnCostByEdge(g.e4to1, g.e1to0, 1, float64(turnCost40))
	f.setTurnCostByEdge(g.e1to2, g.e2to3, 2, float64(turnCost13))
	f.setTurnCostByEdge(g.e0to2, g.e2to3, 2, float64(turnCost03))
	f.freeze()
	f.setMaxLevelOnAllNodes()
	return g
}

func TestEdgeBasedContractNode_detour_multipleInOut_needsShortcut(t *testing.T) {
	f := newEBCTestFixture(t)
	g := newGraphWithDetourMultipleInOutEdges(f, 0, 0, 0, 1, 3)
	f.contractNodes(0, 2, 5, 6, 7, 1, 3, 4)
	ebcCheckShortcuts(t, f.chStore, f.createShortcutFromEdges(1, 4, g.e1to0, g.e0to4, 7))
}

func TestEdgeBasedContractNode_detour_multipleInOut_noShortcuts(t *testing.T) {
	f := newEBCTestFixture(t)
	_ = newGraphWithDetourMultipleInOutEdges(f, 0, 0, 0, 0, 0)
	f.contractNodes(0, 2, 5, 6, 7, 1, 3, 4)
	ebcCheckShortcuts(t, f.chStore)
}

func TestEdgeBasedContractNode_detour_multipleInOut_restrictedIn(t *testing.T) {
	f := newEBCTestFixture(t)
	_ = newGraphWithDetourMultipleInOutEdges(f, 0, ebcMaxCost, 0, ebcMaxCost, 0)
	f.contractNodes(0, 2, 5, 6, 7, 1, 3, 4)
	ebcCheckShortcuts(t, f.chStore)
}

// 5   3   7
//  \ / \ /
// 2-1-0-4-6
type graphWithDetourMultipleInOutEdges struct {
	e5to1 util.EdgeIteratorState
	e2to1 util.EdgeIteratorState
	e1to3 util.EdgeIteratorState
	e3to4 util.EdgeIteratorState
	e1to0 util.EdgeIteratorState
	e0to4 util.EdgeIteratorState
	e4to6 util.EdgeIteratorState
	e4to7 util.EdgeIteratorState
}

func newGraphWithDetourMultipleInOutEdges(f *ebcTestFixture, turnCost20, turnCost50, turnCost23, turnCost53, turnCost36 int) *graphWithDetourMultipleInOutEdges {
	g := &graphWithDetourMultipleInOutEdges{}
	g.e5to1 = f.graph.Edge(5, 1).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e2to1 = f.graph.Edge(2, 1).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e1to3 = f.graph.Edge(1, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e3to4 = f.graph.Edge(3, 4).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e1to0 = f.graph.Edge(1, 0).SetDistance(50).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e0to4 = f.graph.Edge(0, 4).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e4to6 = f.graph.Edge(4, 6).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e4to7 = f.graph.Edge(4, 7).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.setTurnCostByEdge(g.e1to3, g.e3to4, 3, 2)
	f.setTurnCostByEdge(g.e2to1, g.e1to0, 1, float64(turnCost20))
	f.setTurnCostByEdge(g.e2to1, g.e1to3, 1, float64(turnCost23))
	f.setTurnCostByEdge(g.e5to1, g.e1to0, 1, float64(turnCost50))
	f.setTurnCostByEdge(g.e5to1, g.e1to3, 1, float64(turnCost53))
	f.setTurnCostByEdge(g.e3to4, g.e4to6, 4, float64(turnCost36))
	f.freeze()
	f.setMaxLevelOnAllNodes()
	return g
}

func TestEdgeBasedContractNode_loopAvoidance_loopNecessary(t *testing.T) {
	f := newEBCTestFixture(t)
	g := newGraphWithLoop(f, 7)
	f.contractNodes(0, 1, 3, 4, 5, 2)
	numEdges := 6
	ebcCheckShortcuts(t, f.chStore,
		f.createShortcutFromEdgesDir(1, 2, g.e2to0, g.e0to1, 3, false, true),
		createShortcutRawDir(2, 2, g.e2to0.GetEdgeKey(), g.e1to2.GetEdgeKey(), numEdges, g.e1to2.GetEdge(), 4, true, false),
	)
}

func TestEdgeBasedContractNode_loopAvoidance_loopAvoidable(t *testing.T) {
	f := newEBCTestFixture(t)
	g := newGraphWithLoop(f, 3)
	f.contractNodes(0, 1, 3, 4, 5, 2)
	ebcCheckShortcuts(t, f.chStore,
		f.createShortcutFromEdgesDir(1, 2, g.e2to0, g.e0to1, 3, false, true),
	)
}

//   0 - 1
//    \ /
// 3 - 2 - 4
//     |
//     5
type graphWithLoop struct {
	e0to1 util.EdgeIteratorState
	e1to2 util.EdgeIteratorState
	e2to0 util.EdgeIteratorState
	e3to2 util.EdgeIteratorState
	e2to4 util.EdgeIteratorState
	e5to2 util.EdgeIteratorState
}

func newGraphWithLoop(f *ebcTestFixture, turnCost34 int) *graphWithLoop {
	g := &graphWithLoop{}
	g.e0to1 = f.graph.Edge(0, 1).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e1to2 = f.graph.Edge(1, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e2to0 = f.graph.Edge(2, 0).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e3to2 = f.graph.Edge(3, 2).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e2to4 = f.graph.Edge(2, 4).SetDistance(50).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.e5to2 = f.graph.Edge(5, 2).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.setTurnCostByEdge(g.e3to2, g.e2to4, 2, float64(turnCost34))
	f.freeze()
	f.setMaxLevelOnAllNodes()
	return g
}

func TestEdgeBasedContractNode_witnessPathsAreFound(t *testing.T) {
	f := newEBCTestFixture(t)
	//         2 ----- 7 - 10
	//       / |       |
	// 0 - 1   3 - 4   |
	//     |   |      /
	//     5 - 9 ----
	f.graph.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 3).SetDistance(50).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(5, 9).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(9, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 7).SetDistance(60).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(9, 7).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(7, 10).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(2, 0, 10, 4, 1, 5, 7, 9, 3)
	ebcCheckShortcuts(t, f.chStore)
}

func TestEdgeBasedContractNode_noUnnecessaryShortcut_witnessPathOfEqualWeight(t *testing.T) {
	// @RepeatedTest(10) -> run 10 times
	for range 10 {
		f := newEBCTestFixture(t)
		// 0 -> 1 -> 5 <_
		//      v    v   \
		//      2 -> 3 -> 4
		f.graph.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
		f.graph.Edge(1, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
		f.graph.Edge(1, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
		e2to3 := f.graph.Edge(2, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
		e3to4 := f.graph.Edge(3, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
		f.graph.Edge(4, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
		e5to3 := f.graph.Edge(5, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
		f.freeze()
		f.setMaxLevelOnAllNodes()
		f.contractNodes(3, 2, 0, 1, 5, 4)
		ebcCheckShortcuts(t, f.chStore,
			f.createShortcutFromEdges(2, 4, e2to3, e3to4, 2),
			f.createShortcutFromEdges(5, 4, e5to3, e3to4, 2),
		)
	}
}

func TestEdgeBasedContractNode_noUnnecessaryShortcut_differentWitnessesForDifferentOutEdges(t *testing.T) {
	f := newEBCTestFixture(t)
	//         /--> 2 ---\
	//        /           \
	// 0 --> 1 ---> 3 ---> 5 --> 6
	//        \           /
	//         \--> 4 ---/
	f.graph.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	// bidirectional
	f.graph.Edge(2, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(3, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	// bidirectional
	f.graph.Edge(4, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(5, 6).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(3, 0, 6, 1, 2, 5, 4)
	ebcCheckShortcuts(t, f.chStore)
}

func TestEdgeBasedContractNode_noUnnecessaryShortcut_differentInitialEntriesForDifferentInEdges(t *testing.T) {
	f := newEBCTestFixture(t)
	//         /--- 2 ->-\
	//        /           \
	// 0 --> 1 ---> 3 ---> 5 --> 6
	//        \           /
	//         \--- 4 ->-/
	f.graph.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	// bidirectional
	f.graph.Edge(1, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(1, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	// bidirectional
	f.graph.Edge(1, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(2, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(4, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(5, 6).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(3, 0, 6, 1, 2, 5, 4)
	ebcCheckShortcuts(t, f.chStore)
}

func TestEdgeBasedContractNode_bidirectional_edge_at_fromNode(t *testing.T) {
	for _, edge1to2bidirectional := range []bool{true, false} {
		name := "bidirectional"
		if !edge1to2bidirectional {
			name = "unidirectional"
		}
		t.Run(name, func(t *testing.T) {
			f := newEBCTestFixture(t)
			// 0 -> 1 <-> 5
			//      v     v
			//      2 --> 3 -> 4
			f.graph.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			bwd := float64(0)
			if edge1to2bidirectional {
				bwd = 10
			}
			f.graph.Edge(1, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, bwd)
			f.graph.Edge(2, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(3, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.graph.Edge(1, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
			f.graph.Edge(5, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
			f.freeze()
			f.setMaxLevelOnAllNodes()
			f.contractNodes(2, 0, 1, 5, 4, 3)
			// we might come from (5->1) so we still need a way back to (3->4) -> we need a shortcut
			expectedShortcut := createShortcutRaw(1, 3, 2, 4, 1, 2, 2)
			ebcCheckShortcuts(t, f.chStore, expectedShortcut)
		})
	}
}

func TestEdgeBasedContractNode_bidirectional_edge_at_fromNode_going_to_node(t *testing.T) {
	f := newEBCTestFixture(t)
	// 0 -> 1 <-> 5
	//      v     v
	//      2 --> 3 -> 4
	f.graph.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(5, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(5, 0, 4, 1, 2, 3)
	// wherever we come from we can always go via node 2 -> no shortcut needed
	ebcCheckShortcuts(t, f.chStore)
}

func TestEdgeBasedNodeContraction_directWitness(t *testing.T) {
	f := newEBCTestFixture(t)
	// 0 -> 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8
	//     /      \                 /      \
	//10 ->        ------> 9 ------>        -> 11
	f.graph.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(4, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(5, 6).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(6, 7).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(7, 8).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 9).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(9, 6).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(10, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(7, 11).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(2, 6, 3, 5, 4, 0, 8, 10, 11, 1, 7, 9)
	ebcCheckShortcuts(t, f.chStore,
		createShortcutRawDir(3, 1, 2, 4, 1, 2, 2, false, true),
		createShortcutRawDir(1, 9, 2, 16, 1, 8, 2, true, false),
		createShortcutRawDir(5, 7, 10, 12, 5, 6, 2, true, false),
		createShortcutRawDir(7, 9, 18, 12, 9, 6, 2, false, true),
		createShortcutRawDir(4, 1, 2, 6, 12, 3, 3, false, true),
		createShortcutRawDir(4, 7, 8, 12, 4, 13, 3, true, false),
	)
}

func TestEdgeBasedNodeContraction_witnessBetterBecauseOfTurnCostAtTargetNode(t *testing.T) {
	f := newEBCTestFixture(t)
	// 0 -> 1 -> 2 -> 3 -> 4
	//       \       /
	//        -- 5 ->
	f.graph.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 5).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(5, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.setTurnCost(2, 3, 4, 5)
	f.setTurnCost(5, 3, 4, 2)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(2, 0, 4, 1, 3, 5)
	ebcCheckShortcuts(t, f.chStore)
}

func TestEdgeBasedNodeContraction_letShortcutsWitnessEachOther_twoIn(t *testing.T) {
	f := newEBCTestFixture(t)
	// 0 -> 1 -> 2 -> 3 -> 4 -> 5
	//       \        |
	//        ------->|
	f.graph.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 3).SetDistance(40).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(4, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)

	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(3, 0, 5, 1, 4, 2)
	ebcCheckShortcuts(t, f.chStore,
		createShortcutRawDir(4, 2, 4, 6, 2, 3, 2, false, true),
	)
}

func TestEdgeBasedNodeContraction_letShortcutsWitnessEachOther_twoOut(t *testing.T) {
	f := newEBCTestFixture(t)
	// 0 -> 1 -> 2 -> 3 -> 4 -> 5
	//           |        /
	//           ------->
	f.graph.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(4, 5).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 4).SetDistance(40).SetDecimalBothDir(f.speedEnc, 10, 0)

	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(2, 0, 5, 1, 4, 3)
	ebcCheckShortcuts(t, f.chStore,
		createShortcutRaw(1, 3, 2, 4, 1, 2, 2),
	)
}

func TestEdgeBasedNodeContraction_parallelEdges_onlyOneLoopShortcutNeeded(t *testing.T) {
	f := newEBCTestFixture(t)
	//  /--\
	// 0 -- 1 -- 2
	edge0 := f.graph.Edge(0, 1).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 10)
	edge1 := f.graph.Edge(1, 0).SetDistance(40).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(1, 2).SetDistance(50).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.setTurnCostByEdge(edge0, edge1, 0, 1)
	f.setTurnCostByEdge(edge1, edge0, 0, 2)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(0, 2, 1)
	// it is sufficient to be able to travel the 1-0-1 loop in one (the cheaper) direction
	ebcCheckShortcuts(t, f.chStore,
		createShortcutRaw(1, 1, 1, 3, 0, 1, 7),
	)
}

func TestEdgeBasedNodeContraction_duplicateEdge_severalLoops(t *testing.T) {
	f := newEBCTestFixture(t)
	// 5 -- 4 -- 3 -- 1
	// |\   |
	// | \  /
	// -- 2
	f.graph.Edge(1, 3).SetDistance(470).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(2, 4).SetDistance(190).SetDecimalBothDir(f.speedEnc, 10, 10)
	e2 := f.graph.Edge(2, 5).SetDistance(380).SetDecimalBothDir(f.speedEnc, 10, 10)
	e3 := f.graph.Edge(2, 5).SetDistance(570).SetDecimalBothDir(f.speedEnc, 10, 10) // note there is a duplicate edge here (with different weight)
	f.graph.Edge(3, 4).SetDistance(100).SetDecimalBothDir(f.speedEnc, 10, 10)
	e5 := f.graph.Edge(4, 5).SetDistance(560).SetDecimalBothDir(f.speedEnc, 10, 10)

	f.setTurnCostByEdge(e3, e2, 5, 4)
	f.setTurnCostByEdge(e2, e3, 5, 5)
	f.setTurnCostByEdge(e5, e3, 5, 3)
	f.setTurnCostByEdge(e3, e5, 5, 2)
	f.setTurnCostByEdge(e2, e5, 5, 2)
	f.setTurnCostByEdge(e5, e2, 5, 1)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(4, 5, 1, 3, 2)
	ebcCheckNumShortcuts(t, f.chStore, 11)
	ebcCheckShortcuts(t, f.chStore,
		// from node 4 contraction
		createShortcutRawDir(5, 3, 11, 9, 5, 4, 66, true, false),
		createShortcutRawDir(5, 3, 8, 10, 4, 5, 66, false, true),
		createShortcutRawDir(3, 2, 2, 9, 1, 4, 29, false, true),
		createShortcutRawDir(3, 2, 8, 3, 4, 1, 29, true, false),
		createShortcutRawDir(5, 2, 2, 10, 1, 5, 75, false, true),
		createShortcutRawDir(5, 2, 11, 3, 5, 1, 75, true, false),
		// from node 5 contraction
		createShortcutRawDir(2, 2, 6, 5, 3, 2, 99, true, false),
		createShortcutRawDir(2, 2, 6, 3, 3, 6, 134, true, false),
		createShortcutRawDir(2, 2, 2, 5, 8, 2, 114, true, false),
		createShortcutRawDir(3, 2, 4, 9, 2, 7, 106, false, true),
		createShortcutRawDir(3, 2, 8, 5, 9, 2, 105, true, false),
	)
}

func TestEdgeBasedNodeContraction_tripleConnection(t *testing.T) {
	f := newEBCTestFixture(t)
	f.graph.Edge(0, 1).SetDistance(10.0).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(0, 1).SetDistance(20.0).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(0, 1).SetDistance(35.0).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(1, 0)
	ebcCheckShortcuts(t, f.chStore,
		createShortcutRaw(0, 0, 2, 5, 1, 2, 5.5),
		createShortcutRaw(0, 0, 0, 5, 0, 2, 4.5),
		createShortcutRaw(0, 0, 0, 3, 0, 1, 3.0),
	)
}

func TestEdgeBasedNodeContraction_fromAndToNodesEqual(t *testing.T) {
	f := newEBCTestFixture(t)
	// 0 -> 1 -> 3
	//     / \
	//    v   ^
	//     \ /
	//      2
	f.graph.Edge(0, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 3).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(2, 0, 1, 3)
	ebcCheckShortcuts(t, f.chStore)
}

func TestEdgeBasedNodeContraction_node_in_loop(t *testing.T) {
	f := newEBCTestFixture(t)
	//      2
	//     /|
	//  0-4-3
	//    |
	//    1
	f.graph.Edge(0, 4).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(4, 3).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(3, 2).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(2, 4).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(4, 1).SetDistance(10).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.freeze()
	f.setMaxLevelOnAllNodes()

	// enforce loop (going counter-clockwise)
	f.setRestriction(0, 4, 1)
	f.setTurnCost(4, 2, 3, 4)
	f.setTurnCost(3, 2, 4, 2)
	f.contractNodes(2, 0, 1, 4, 3)
	ebcCheckShortcuts(t, f.chStore,
		createShortcutRawDir(4, 3, 7, 5, 3, 2, 6, true, false),
		createShortcutRawDir(4, 3, 4, 6, 2, 3, 4, false, true),
	)
}

func TestEdgeBasedFindPath_finiteUTurnCost(t *testing.T) {
	f := newEBCTestFixture(t)
	//   1
	//   |
	// 0-3-4
	//   |/
	//   2
	f.graph.Edge(0, 3).SetDistance(1000).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 4).SetDistance(1000).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(4, 2).SetDistance(5000).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 3).SetDistance(2000).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 1).SetDistance(1000).SetDecimalBothDir(f.speedEnc, 10, 0)
	// Use config index 1 (uTurnCosts=60)
	f.freezeWithConfig(1)
	f.setMaxLevelOnAllNodes()
	f.setRestriction(0, 3, 1)
	f.contractNodes(4, 0, 1, 2, 3)
	ebcCheckShortcuts(t, f.chStore,
		createShortcutRawDir(2, 3, 2, 4, 1, 2, 600, false, true),
		createShortcutRawDir(3, 3, 2, 3, 1, 1, 260, true, false),
	)
}

func TestEdgeBasedNodeContraction_turnRestrictionAndLoop(t *testing.T) {
	f := newEBCTestFixture(t)
	//  /\    /<-3
	// 0  1--2
	//  \/    \->4
	f.graph.Edge(0, 1).SetDistance(50).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(0, 1).SetDistance(60).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(1, 2).SetDistance(20).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(3, 2).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 4).SetDistance(30).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.setRestriction(3, 2, 4)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(0, 3, 4, 2, 1)
	ebcCheckNumShortcuts(t, f.chStore, 1)
}

func TestEdgeBasedNodeContraction_minorWeightDeviation(t *testing.T) {
	f := newEBCTestFixture(t)
	// 0 -> 1 -> 2 -> 3 -> 4
	f.graph.Edge(0, 1).SetDistance(514.01).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(1, 2).SetDistance(700.41).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 3).SetDistance(758.06).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(3, 4).SetDistance(050.03).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	f.contractNodes(2, 0, 1, 3, 4)
	ebcCheckShortcuts(t, f.chStore,
		createShortcutRaw(1, 3, 2, 4, 1, 2, 145.847),
	)
}

func TestEdgeBasedNodeContraction_numPolledEdges(t *testing.T) {
	f := newEBCTestFixture(t)
	//           1<-6
	//           |
	// 0 -> 3 -> 2 <-> 4 -> 5
	//  \---<----|
	f.graph.Edge(3, 2).SetDistance(710.203000).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(0, 3).SetDistance(790.003000).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 0).SetDistance(210.328000).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 4).SetDistance(160.499000).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(4, 2).SetDistance(160.487000).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(6, 1).SetDistance(550.603000).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(2, 1).SetDistance(330.453000).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.graph.Edge(4, 5).SetDistance(290.665000).SetDecimalBothDir(f.speedEnc, 10, 0)
	f.freeze()
	f.setMaxLevelOnAllNodes()
	nc := f.createNodeContractor()
	nc.ContractNode(0)
	require.Greater(t, nc.GetNumPolledEdges(), int64(0), "no polled edges, something is wrong")
	require.LessOrEqual(t, nc.GetNumPolledEdges(), int64(8), "too many edges polled: %d", nc.GetNumPolledEdges())
}

func TestEdgeBasedIssue_2564(t *testing.T) {
	f := newEBCTestFixture(t)
	// 0-1-2-3-4-5
	f.graph.Edge(0, 1).SetDistance(1000).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(1, 2).SetDistance(73.36).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(2, 3).SetDistance(101.61).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(3, 4).SetDistance(0).SetDecimalBothDir(f.speedEnc, 10, 10)
	f.graph.Edge(4, 5).SetDistance(1000).SetDecimalBothDir(f.speedEnc, 10, 10)
	// Use config index 2 (uTurnCosts=0)
	f.freezeWithConfig(2)
	f.setMaxLevelOnAllNodes()
	f.contractNodes(0, 5, 2, 1, 3, 4)
	ebcCheckShortcuts(t, f.chStore,
		createShortcutRawDir(1, 3, 2, 4, 1, 2, 17.497, true, false),
		createShortcutRawDir(1, 3, 5, 3, 2, 1, 17.497, false, true),
	)
}
