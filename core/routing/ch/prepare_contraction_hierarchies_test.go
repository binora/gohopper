package ch

import (
	"math"
	"testing"

	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"

	"github.com/stretchr/testify/assert"
)

// --- test helpers ---

func pchCreateGraph(speedEnc ev.DecimalEncodedValue) *storage.BaseGraph {
	em := routingutil.Start().Add(speedEnc).Build()
	return storage.NewBaseGraphBuilder(em.BytesForFlags).CreateGraph()
}

func pchCreatePrepare(g *storage.BaseGraph, chConfig *CHConfig) *PrepareContractionHierarchies {
	if !g.IsFrozen() {
		g.Freeze()
	}
	return FromGraph(g, chConfig)
}

func pchUseNodeOrdering(prepare *PrepareContractionHierarchies, ordering []int) {
	prepare.UseFixedNodeOrdering(NodeOrderingFromArray(ordering...))
}

// initExampleGraph creates:
//
//	5-1-----2
//	   \ __/|
//	    0   |
//	   /    |
//	  4-----3
func initExampleGraph(g *storage.BaseGraph, speedEnc ev.DecimalEncodedValue) {
	g.Edge(0, 1).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(0, 2).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(0, 4).SetDistance(3).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(1, 2).SetDistance(3).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(2, 3).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(4, 3).SetDistance(2).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(5, 1).SetDistance(2).SetDecimalBothDir(speedEnc, 60, 60)
}

// initShortcutsGraph creates the prepare-routing.svg graph (17 nodes, 22 edges).
func initShortcutsGraph(g *storage.BaseGraph, speedEnc ev.DecimalEncodedValue) {
	g.Edge(0, 1).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(0, 2).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(1, 2).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(2, 3).SetDistance(1.5).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(1, 4).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(2, 9).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(9, 3).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(10, 3).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(4, 5).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(5, 6).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(6, 7).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(7, 8).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(8, 9).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(4, 11).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(9, 14).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(10, 14).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(11, 12).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(12, 15).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(12, 13).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(13, 16).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(15, 16).SetDistance(2).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(14, 16).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
}

func pchGetAdjs(iter storage.RoutingCHEdgeIterator) []int {
	var result []int
	for iter.Next() {
		result = append(result, iter.GetAdjNode())
	}
	return result
}

// --- tests ---

func TestPrepareContractionHierarchies_ReturnsCorrectWeighting(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	g := pchCreateGraph(speedEnc)
	w := weighting.NewSpeedWeighting(speedEnc)
	chConfig := NewCHConfigNodeBased("c", w)
	prepare := pchCreatePrepare(g, chConfig)
	result := prepare.DoWork()
	assert.Same(t, w, result.GetCHConfig().GetWeighting())
}

func TestPrepareContractionHierarchies_AddShortcuts(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	g := pchCreateGraph(speedEnc)
	initExampleGraph(g, speedEnc)
	w := weighting.NewSpeedWeighting(speedEnc)
	chConfig := NewCHConfigNodeBased("c", w)
	prepare := pchCreatePrepare(g, chConfig)
	pchUseNodeOrdering(prepare, []int{5, 3, 4, 0, 1, 2})
	result := prepare.DoWork()
	assert.Equal(t, int64(2), result.GetShortcuts())
}

func TestPrepareContractionHierarchies_MoreComplexGraph(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	g := pchCreateGraph(speedEnc)
	initShortcutsGraph(g, speedEnc)
	w := weighting.NewSpeedWeighting(speedEnc)
	chConfig := NewCHConfigNodeBased("c", w)
	prepare := pchCreatePrepare(g, chConfig)
	pchUseNodeOrdering(prepare, []int{0, 5, 6, 7, 8, 10, 11, 13, 15, 1, 3, 9, 14, 16, 12, 4, 2})
	result := prepare.DoWork()
	assert.Equal(t, int64(7), result.GetShortcuts())
}

func TestPrepareContractionHierarchies_CircleBug(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	g := pchCreateGraph(speedEnc)
	//  /--1
	// -0--/
	//  |
	g.Edge(0, 1).SetDistance(10).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(0, 1).SetDistance(4).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(0, 2).SetDistance(10).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(0, 3).SetDistance(10).SetDecimalBothDir(speedEnc, 60, 60)
	w := weighting.NewSpeedWeighting(speedEnc)
	chConfig := NewCHConfigNodeBased("c", w)
	prepare := pchCreatePrepare(g, chConfig)
	result := prepare.DoWork()
	assert.Equal(t, int64(0), result.GetShortcuts())
}

func TestPrepareContractionHierarchies_Bug178(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	g := pchCreateGraph(speedEnc)
	// 5--------6__
	// |        |  \
	// 0-1->-2--3--4
	//   \-<-/
	g.Edge(1, 2).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(2, 1).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(5, 0).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(5, 6).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(0, 1).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(2, 3).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(3, 4).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(6, 3).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	w := weighting.NewSpeedWeighting(speedEnc)
	chConfig := NewCHConfigNodeBased("c", w)
	prepare := pchCreatePrepare(g, chConfig)
	pchUseNodeOrdering(prepare, []int{4, 1, 2, 0, 5, 6, 3})
	result := prepare.DoWork()
	assert.Equal(t, int64(2), result.GetShortcuts())
}

func TestPrepareContractionHierarchies_Bits(t *testing.T) {
	fromNode := int64(math.MaxInt32) / 3 * 2
	endNode := int64(math.MaxInt32) / 37 * 17

	edgeId := fromNode<<32 | endNode
	assert.Equal(t,
		util.BitLE.ToBitString(edgeId, 64),
		util.BitLE.ToLastBitString(fromNode, 32)+util.BitLE.ToLastBitString(endNode, 32),
	)
}

func TestPrepareContractionHierarchies_Disconnects(t *testing.T) {
	//            4
	//            v
	//            0
	//            v
	//  8 -> 3 -> 6 -> 1 -> 5
	//            v
	//            2
	//            v
	//            7
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	g := pchCreateGraph(speedEnc)
	g.Edge(8, 3).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(3, 6).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(6, 1).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(1, 5).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(4, 0).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(0, 6).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(6, 2).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(2, 7).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Freeze()

	w := weighting.NewSpeedWeighting(speedEnc)
	chConfig := NewCHConfigNodeBased("c", w)
	prepare := pchCreatePrepare(g, chConfig).
		UseFixedNodeOrdering(IdentityNodeOrdering(g.GetNodes()))
	result := prepare.DoWork()

	routingCHGraph := storage.NewRoutingCHGraph(g, result.GetCHStorage(), w)
	outExplorer := routingCHGraph.CreateOutEdgeExplorer()
	inExplorer := routingCHGraph.CreateInEdgeExplorer()

	// shortcuts leading to or coming from lower level nodes are not visible
	// so far we still receive base graph edges leading to or coming from lower level nodes though
	assert.Equal(t, []int{7, 2, 1}, pchGetAdjs(outExplorer.SetBaseNode(6)))
	assert.Equal(t, []int{8, 0, 3}, pchGetAdjs(inExplorer.SetBaseNode(6)))
	assert.Equal(t, []int{6, 0}, pchGetAdjs(outExplorer.SetBaseNode(4)))
	assert.Equal(t, []int{6, 1}, pchGetAdjs(inExplorer.SetBaseNode(5)))
	assert.Equal(t, []int{8, 2}, pchGetAdjs(inExplorer.SetBaseNode(7)))
	assert.Equal(t, []int{3}, pchGetAdjs(outExplorer.SetBaseNode(8)))
	assert.Equal(t, []int(nil), pchGetAdjs(inExplorer.SetBaseNode(8)))
}

// TODO: port these tests when CH routing algorithm is available:
// - TestPrepareContractionHierarchies_DirectedGraph
// - TestPrepareContractionHierarchies_DirectedGraph2
// - TestPrepareContractionHierarchies_RoundaboutUnpacking
// - TestPrepareContractionHierarchies_StallOnDemandViaVirtualNode
// - TestPrepareContractionHierarchies_MultiplePreparationsIdenticalView
// - TestPrepareContractionHierarchies_MultiplePreparationsDifferentView
// - TestPrepareContractionHierarchies_ReusingNodeOrdering
