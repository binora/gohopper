package ch

import (
	"math"
	"math/rand"
	"testing"

	"gohopper/core/routing"
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

// initDirected2 creates:
//
//	0-1-.....-9-10
//	|         ^   \
//	|         |    |
//	17-16-...-11<-/
func initDirected2(g *storage.BaseGraph, speedEnc ev.DecimalEncodedValue) {
	g.Edge(0, 1).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(1, 2).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(2, 3).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(3, 4).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(4, 5).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(5, 6).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(6, 7).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(7, 8).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(8, 9).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(9, 10).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(10, 11).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(11, 12).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(11, 9).SetDistance(3).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(12, 13).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(13, 14).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(14, 15).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(15, 16).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(16, 17).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(17, 0).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
}

func initRoundaboutGraph(g *storage.BaseGraph, speedEnc ev.DecimalEncodedValue) {
	g.Edge(16, 0).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(0, 9).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(0, 17).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(9, 10).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(10, 11).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(11, 28).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(28, 29).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(29, 30).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(30, 31).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(31, 4).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(17, 1).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(15, 1).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(14, 1).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(14, 18).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(18, 19).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(19, 20).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(20, 15).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(19, 21).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(21, 16).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(1, 2).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(2, 3).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(3, 4).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(4, 5).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(5, 6).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(6, 7).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(7, 13).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(13, 12).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(12, 4).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(7, 8).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(8, 22).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(22, 23).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(23, 24).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(24, 25).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(25, 27).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(27, 5).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(25, 26).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(26, 25).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
}

func pchGetAdjs(iter storage.RoutingCHEdgeIterator) []int {
	var result []int
	for iter.Next() {
		result = append(result, iter.GetAdjNode())
	}
	return result
}

func pchGetEdge(g *storage.BaseGraph, from, to int) util.EdgeIteratorState {
	iter := g.CreateEdgeExplorer(routingutil.AllEdges).SetBaseNode(from)
	for iter.Next() {
		if iter.GetAdjNode() == to {
			return iter
		}
	}
	panic("edge not found")
}

func pchCheckPath(t *testing.T, g *storage.BaseGraph, chConfig *CHConfig, expShortcuts int64, expDistance float64, expNodes []int, nodeOrdering []int) {
	t.Helper()
	prepare := pchCreatePrepare(g, chConfig)
	pchUseNodeOrdering(prepare, nodeOrdering)
	result := prepare.DoWork()
	assert.Equal(t, expShortcuts, result.GetShortcuts(), chConfig.GetName())
	routingCHGraph := storage.NewRoutingCHGraph(g, result.GetCHStorage(), chConfig.GetWeighting())
	path := NewCHRoutingAlgorithmFactory(routingCHGraph).CreateAlgo(nil).CalcPath(3, 12)
	assert.InDelta(t, expDistance, path.Distance, 1e-5, path.String())
	assert.Equal(t, expNodes, path.CalcNodes(), path.String())
}

func pchBuildRandomConnectedGraph(g *storage.BaseGraph, rnd *rand.Rand, numNodes int) {
	for i := 0; i < numNodes-1; i++ {
		g.Edge(i, i+1).SetDistance(1 + rnd.Float64()*9)
	}
	g.Edge(numNodes-1, 0).SetDistance(1 + rnd.Float64()*9)
	for i := 0; i < numNodes/3; i++ {
		a := rnd.Intn(numNodes)
		b := rnd.Intn(numNodes)
		if a == b {
			b = (b + 1) % numNodes
		}
		g.Edge(a, b).SetDistance(1 + rnd.Float64()*9)
	}
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

func TestPrepareContractionHierarchies_DirectedGraph(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	g := pchCreateGraph(speedEnc)
	g.Edge(5, 4).SetDistance(3).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(4, 5).SetDistance(10).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(2, 4).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(5, 2).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(3, 5).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(4, 3).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Freeze()
	assert.Equal(t, 6, g.GetEdges())

	w := weighting.NewSpeedWeighting(speedEnc)
	chConfig := NewCHConfigNodeBased("c", w)
	prepare := pchCreatePrepare(g, chConfig)
	result := prepare.DoWork()
	assert.Equal(t, int64(2), result.GetShortcuts())

	routingCHGraph := storage.NewRoutingCHGraph(g, result.GetCHStorage(), w)
	assert.Equal(t, 6+2, routingCHGraph.GetEdges())
	path := NewCHRoutingAlgorithmFactory(routingCHGraph).CreateAlgo(nil).CalcPath(4, 2)
	assert.InDelta(t, 3, path.Distance, 1e-6)
	assert.Equal(t, []int{4, 3, 5, 2}, path.CalcNodes())
}

func TestPrepareContractionHierarchies_DirectedGraph2(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	g := pchCreateGraph(speedEnc)
	initDirected2(g, speedEnc)
	oldCount := g.GetEdges()
	assert.Equal(t, 19, oldCount)

	w := weighting.NewSpeedWeighting(speedEnc)
	chConfig := NewCHConfigNodeBased("c", w)
	prepare := pchCreatePrepare(g, chConfig)
	pchUseNodeOrdering(prepare, []int{10, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 17, 11, 12, 13, 14, 15, 16})
	result := prepare.DoWork()
	assert.Equal(t, oldCount, g.GetEdges())
	assert.Equal(t, oldCount, g.GetAllEdges().Length())
	assert.Equal(t, int64(9), result.GetShortcuts())

	routingCHGraph := storage.NewRoutingCHGraph(g, result.GetCHStorage(), w)
	assert.Equal(t, oldCount+9, routingCHGraph.GetEdges())
	path := NewCHRoutingAlgorithmFactory(routingCHGraph).CreateAlgo(nil).CalcPath(0, 10)
	assert.InDelta(t, 10, path.Distance, 1e-6)
	assert.Equal(t, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, path.CalcNodes())
}

func TestPrepareContractionHierarchies_RoundaboutUnpacking(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	g := pchCreateGraph(speedEnc)
	initRoundaboutGraph(g, speedEnc)
	oldCount := g.GetEdges()

	w := weighting.NewSpeedWeighting(speedEnc)
	chConfig := NewCHConfigNodeBased("c", w)
	prepare := pchCreatePrepare(g, chConfig)
	pchUseNodeOrdering(prepare, []int{26, 6, 12, 13, 2, 3, 8, 9, 10, 11, 14, 15, 16, 17, 18, 20, 21, 23, 24, 25, 19, 22, 27, 5, 29, 30, 31, 28, 7, 1, 0, 4})
	result := prepare.DoWork()
	assert.Equal(t, oldCount, g.GetEdges())

	routingCHGraph := storage.NewRoutingCHGraph(g, result.GetCHStorage(), w)
	assert.Equal(t, oldCount, routingCHGraph.GetBaseGraph().GetEdges())
	assert.Equal(t, oldCount+23, routingCHGraph.GetEdges())
	path := NewCHRoutingAlgorithmFactory(routingCHGraph).CreateAlgo(nil).CalcPath(4, 7)
	assert.Equal(t, []int{4, 5, 6, 7}, path.CalcNodes())
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

func TestPrepareContractionHierarchies_MultiplePreparationsIdenticalView(t *testing.T) {
	carSpeedEnc := ev.NewDecimalEncodedValueImpl("car_speed", 5, 5, true)
	bikeSpeedEnc := ev.NewDecimalEncodedValueImpl("bike_speed", 4, 2, true)
	em := routingutil.Start().Add(carSpeedEnc).Add(bikeSpeedEnc).Build()

	carProfile := NewCHConfigNodeBased("c1", weighting.NewSpeedWeighting(carSpeedEnc))
	bikeProfile := NewCHConfigNodeBased("c2", weighting.NewSpeedWeighting(bikeSpeedEnc))

	g := storage.NewBaseGraphBuilder(em.BytesForFlags).CreateGraph()
	initShortcutsGraph(g, carSpeedEnc)
	iter := g.GetAllEdges()
	for iter.Next() {
		iter.SetDecimalBothDir(bikeSpeedEnc, 18, 18)
	}
	g.Freeze()

	ordering := []int{0, 5, 6, 7, 8, 10, 11, 13, 15, 1, 3, 9, 14, 16, 12, 4, 2}
	pchCheckPath(t, g, carProfile, 7, 5, []int{3, 9, 14, 16, 13, 12}, ordering)
	pchCheckPath(t, g, bikeProfile, 7, 5, []int{3, 9, 14, 16, 13, 12}, ordering)
}

func TestPrepareContractionHierarchies_MultiplePreparationsDifferentView(t *testing.T) {
	carSpeedEnc := ev.NewDecimalEncodedValueImpl("car_speed", 5, 5, true)
	bikeSpeedEnc := ev.NewDecimalEncodedValueImpl("bike_speed", 4, 2, true)
	em := routingutil.Start().Add(carSpeedEnc).Add(bikeSpeedEnc).Build()

	carConfig := NewCHConfigNodeBased("c1", weighting.NewSpeedWeighting(carSpeedEnc))
	bikeConfig := NewCHConfigNodeBased("c2", weighting.NewSpeedWeighting(bikeSpeedEnc))

	g := storage.NewBaseGraphBuilder(em.BytesForFlags).CreateGraph()
	initShortcutsGraph(g, carSpeedEnc)
	iter := g.GetAllEdges()
	for iter.Next() {
		iter.SetDecimalBothDir(bikeSpeedEnc, 18, 18)
	}
	pchGetEdge(g, 9, 14).SetDecimalBothDir(bikeSpeedEnc, 0, 0)
	g.Freeze()

	pchCheckPath(t, g, carConfig, 7, 5, []int{3, 9, 14, 16, 13, 12}, []int{0, 5, 6, 7, 8, 10, 11, 13, 15, 1, 3, 9, 14, 16, 12, 4, 2})
	pchCheckPath(t, g, bikeConfig, 9, 5, []int{3, 10, 14, 16, 13, 12}, []int{0, 5, 6, 7, 8, 10, 11, 13, 14, 15, 9, 1, 4, 3, 2, 12, 16})
}

func TestPrepareContractionHierarchies_ReusingNodeOrdering(t *testing.T) {
	car1SpeedEnc := ev.NewDecimalEncodedValueImpl("car1_speed", 5, 5, true)
	car2SpeedEnc := ev.NewDecimalEncodedValueImpl("car2_speed", 5, 5, true)
	car1TurnCostEnc := ev.TurnCostCreate("car1", 1)
	car2TurnCostEnc := ev.TurnCostCreate("car2", 1)
	em := routingutil.Start().
		Add(car1SpeedEnc).AddTurnCostEncodedValue(car1TurnCostEnc).
		Add(car2SpeedEnc).AddTurnCostEncodedValue(car2TurnCostEnc).
		Build()
	car1Config := NewCHConfigNodeBased("c1", weighting.NewSpeedWeighting(car1SpeedEnc))
	car2Config := NewCHConfigNodeBased("c2", weighting.NewSpeedWeighting(car2SpeedEnc))
	g := storage.NewBaseGraphBuilder(em.BytesForFlags).CreateGraph()

	numNodes := 500
	numQueries := 40
	rnd := rand.New(rand.NewSource(42))
	pchBuildRandomConnectedGraph(g, rnd, numNodes)
	iter := g.GetAllEdges()
	for iter.Next() {
		car1Fwd := rnd.Float64() * 100
		if rnd.Float64() < 0.05 {
			car1Fwd = 0
		}
		car1Bwd := rnd.Float64() * 100
		if rnd.Float64() < 0.05 {
			car1Bwd = 0
		}
		car2Fwd := rnd.Float64() * 100
		if rnd.Float64() < 0.05 {
			car2Fwd = 0
		}
		car2Bwd := rnd.Float64() * 100
		if rnd.Float64() < 0.05 {
			car2Bwd = 0
		}
		iter.SetDecimalBothDir(car1SpeedEnc, car1Fwd, car1Bwd)
		iter.SetDecimalBothDir(car2SpeedEnc, car2Fwd, car2Bwd)
	}
	g.Freeze()

	car1PCH := FromGraph(g, car1Config)
	resCar1 := car1PCH.DoWork()

	car1CHStore := resCar1.GetCHStorage()
	car2PCH := FromGraph(g, car2Config).
		UseFixedNodeOrdering(NodeOrderingFromFunc(g.GetNodes(), car1CHStore.GetNodeOrderingProvider()))
	resCar2 := car2PCH.DoWork()
	car2CH := storage.NewRoutingCHGraph(g, resCar2.GetCHStorage(), car2Config.GetWeighting())

	assert.True(t, car1CHStore.GetShortcuts() > 0 && resCar2.GetCHStorage().GetShortcuts() > 0)

	for i := 0; i < numQueries; i++ {
		dijkstra := routing.NewDijkstra(g, car2Config.GetWeighting(), routingutil.NodeBased)
		chAlgo := NewCHRoutingAlgorithmFactory(car2CH).CreateAlgo(nil)

		from := rnd.Intn(numNodes)
		to := rnd.Intn(numNodes)
		dijkstraWeight := dijkstra.CalcPath(from, to).Weight
		chWeight := chAlgo.CalcPath(from, to).Weight
		assert.InDelta(t, dijkstraWeight, chWeight, 1e-1)
	}
}

// TODO: port these tests when CH routing algorithm is available:
// - TestPrepareContractionHierarchies_StallOnDemandViaVirtualNode
