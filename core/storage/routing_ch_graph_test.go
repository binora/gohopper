package storage_test

import (
	"math"
	"testing"

	"gohopper/core/routing/ch"
	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- CH test helpers ---

func countCH(iter storage.RoutingCHEdgeIterator) int {
	n := 0
	for iter.Next() {
		n++
	}
	return n
}

func chNeighbors(iter storage.RoutingCHEdgeIterator) map[int]bool {
	set := make(map[int]bool)
	for iter.Next() {
		set[iter.GetAdjNode()] = true
	}
	return set
}

func chEdgeIDs(iter storage.RoutingCHEdgeIterator) map[int]bool {
	set := make(map[int]bool)
	for iter.Next() {
		set[iter.GetEdge()] = true
	}
	return set
}

// newSpeedGraph creates a BaseGraph with a single speed encoder and returns both.
func newSpeedGraph(t *testing.T, speedEnc ev.DecimalEncodedValue, opts ...func(*storage.BaseGraphBuilder)) (*storage.BaseGraph, ev.DecimalEncodedValue) {
	t.Helper()
	em := routingutil.Start().Add(speedEnc).Build()
	builder := storage.NewBaseGraphBuilder(em.BytesForFlags)
	for _, opt := range opts {
		opt(builder)
	}
	return builder.CreateGraph(), speedEnc
}

// newSpeedGraphDefault creates a BaseGraph with the default speed(5,5,false) encoder.
func newSpeedGraphDefault(t *testing.T, opts ...func(*storage.BaseGraphBuilder)) (*storage.BaseGraph, ev.DecimalEncodedValue) {
	t.Helper()
	return newSpeedGraph(t, ev.NewDecimalEncodedValueImpl("speed", 5, 5, false), opts...)
}

// withElevation is a BaseGraphBuilder option that enables elevation support.
func withElevation(b *storage.BaseGraphBuilder) {
	b.SetWithElevation(true)
}

// newNodeBasedCHGraph creates a frozen CH routing graph with identity levels (node-based).
func newNodeBasedCHGraph(t *testing.T, graph *storage.BaseGraph, speedEnc ev.DecimalEncodedValue, profile string) (*storage.RoutingCHGraphImpl, *storage.CHStorage, *storage.CHStorageBuilder) {
	t.Helper()
	w := weighting.NewSpeedWeighting(speedEnc)
	store := storage.CHStorageFromGraph(graph, profile, false)
	chBuilder := storage.NewCHStorageBuilder(store)
	chBuilder.SetIdentityLevels()
	chGraph := storage.NewRoutingCHGraph(graph, store, w)
	return chGraph, store, chBuilder
}

// --- Tests ---

func TestRoutingCHGraph_BaseAndCHEdges(t *testing.T) {
	graph, speedEnc := newSpeedGraphDefault(t)
	graph.Edge(1, 0)
	graph.Edge(8, 9)
	graph.Freeze()

	chGraph, _, chBuilder := newNodeBasedCHGraph(t, graph, speedEnc, "p")

	assert.Equal(t, 1, util.Count(graph.CreateEdgeExplorer(routingutil.AllEdges).SetBaseNode(1)))
	// routing ch graph does not see edges without access
	assert.Equal(t, 0, countCH(chGraph.CreateInEdgeExplorer().SetBaseNode(1)))

	chBuilder.AddShortcutNodeBased(2, 3, ch.ScDirMask, 10, util.NoEdge, util.NoEdge)

	// should be identical to results before we added shortcut
	assert.Equal(t, 1, util.Count(graph.CreateEdgeExplorer(routingutil.AllEdges).SetBaseNode(1)))
	assert.Equal(t, 0, countCH(chGraph.CreateOutEdgeExplorer().SetBaseNode(1)))

	// base graph does not see shortcut
	assert.Equal(t, 0, util.Count(graph.CreateEdgeExplorer(routingutil.AllEdges).SetBaseNode(2)))
	assert.Equal(t, 1, countCH(chGraph.CreateOutEdgeExplorer().SetBaseNode(2)))

	assert.Equal(t, 10, chGraph.GetNodes())
	assert.Equal(t, 2, graph.GetEdges())
	assert.Equal(t, 3, chGraph.GetEdges())
	assert.Equal(t, 1, countCH(chGraph.CreateOutEdgeExplorer().SetBaseNode(2)))
}

func TestRoutingCHGraph_ShortcutConnection(t *testing.T) {
	//   4 ------ 1 > 0
	//            ^ \
	//            3  2
	graph, speedEnc := newSpeedGraphDefault(t)
	graph.Edge(4, 1).SetDistance(30).SetDecimal(speedEnc, 60)
	graph.Freeze()

	w := weighting.NewSpeedWeighting(speedEnc)
	store := storage.CHStorageFromGraph(graph, "ch", false)
	chBuilder := storage.NewCHStorageBuilder(store)
	chBuilder.SetIdentityLevels()
	chBuilder.AddShortcutNodeBased(0, 1, ch.ScBwdDir, 10, 12, 13)
	chBuilder.AddShortcutNodeBased(1, 2, ch.ScDirMask, 10, 10, 11)
	chBuilder.AddShortcutNodeBased(1, 3, ch.ScBwdDir, 10, 14, 15)

	baseExplorer := graph.CreateEdgeExplorer(routingutil.AllEdges)
	lg := storage.NewRoutingCHGraph(graph, store, w)
	chOutExplorer := lg.CreateOutEdgeExplorer()
	chInExplorer := lg.CreateInEdgeExplorer()

	// shortcuts are only visible from the lower level node
	assert.Equal(t, 0, countCH(chOutExplorer.SetBaseNode(2)))
	assert.Equal(t, 0, countCH(chInExplorer.SetBaseNode(2)))

	assert.Equal(t, 2, countCH(chOutExplorer.SetBaseNode(1)))
	assert.Equal(t, 3, countCH(chInExplorer.SetBaseNode(1)))
	assert.Equal(t, util.AsSet(2, 4), chNeighbors(chOutExplorer.SetBaseNode(1)))
	assert.Equal(t, util.AsSet(4), util.GetNeighbors(baseExplorer.SetBaseNode(1)))

	assert.Equal(t, 0, countCH(chOutExplorer.SetBaseNode(3)))
	assert.Equal(t, 0, countCH(chInExplorer.SetBaseNode(3)))

	assert.Equal(t, 0, countCH(chOutExplorer.SetBaseNode(0)))
	assert.Equal(t, 1, countCH(chInExplorer.SetBaseNode(0)))
}

func TestRoutingCHGraph_GetWeight(t *testing.T) {
	graph, speedEnc := newSpeedGraphDefault(t)
	edge1 := graph.Edge(0, 1)
	edge2 := graph.Edge(1, 2)
	graph.Freeze()

	w := weighting.NewSpeedWeighting(speedEnc)
	store := storage.CHStorageFromGraph(graph, "ch", false)
	g := storage.NewRoutingCHGraph(graph, store, w)
	assert.False(t, g.GetEdgeIteratorState(edge1.GetEdge(), math.MinInt32).IsShortcut())
	assert.False(t, g.GetEdgeIteratorState(edge2.GetEdge(), math.MinInt32).IsShortcut())

	chBuilder := storage.NewCHStorageBuilder(store)
	chBuilder.SetIdentityLevels()
	chBuilder.AddShortcutNodeBased(0, 1, ch.ScDirMask, 5, util.NoEdge, util.NoEdge)
	sc1 := g.GetEdgeIteratorState(2, 1)
	assert.Equal(t, 0, sc1.GetBaseNode())
	assert.Equal(t, 1, sc1.GetAdjNode())
	assert.InDelta(t, 5.0, sc1.GetWeight(false), 1e-3)
	assert.True(t, sc1.IsShortcut())
}

func TestRoutingCHGraph_GetWeightAdvancedEncoder(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 4, 2, true)
	graph, _ := newSpeedGraph(t, speedEnc)
	graph.Edge(0, 3)
	graph.Freeze()

	w := weighting.NewSpeedWeighting(speedEnc)
	store := storage.CHStorageFromGraph(graph, "p1", false)
	chBuilder := storage.NewCHStorageBuilder(store)
	chBuilder.SetIdentityLevels()
	sc1 := graph.GetEdges() + chBuilder.AddShortcutNodeBased(0, 1, ch.ScFwdDir, 100.123, util.NoEdge, util.NoEdge)
	lg := storage.NewRoutingCHGraph(graph, store, w)
	assert.Equal(t, 1, lg.GetEdgeIteratorState(sc1, 1).GetAdjNode())
	assert.Equal(t, 0, lg.GetEdgeIteratorState(sc1, 1).GetBaseNode())
	assert.InDelta(t, 100.123, lg.GetEdgeIteratorState(sc1, 1).GetWeight(false), 1e-3)
	assert.InDelta(t, 100.123, lg.GetEdgeIteratorState(sc1, 0).GetWeight(false), 1e-3)

	sc2 := graph.GetEdges() + chBuilder.AddShortcutNodeBased(2, 3, ch.ScDirMask, 1.011011, util.NoEdge, util.NoEdge)
	assert.Equal(t, 3, lg.GetEdgeIteratorState(sc2, 3).GetAdjNode())
	assert.Equal(t, 2, lg.GetEdgeIteratorState(sc2, 3).GetBaseNode())
	assert.InDelta(t, 1.011011, lg.GetEdgeIteratorState(sc2, 2).GetWeight(false), 1e-3)
	assert.InDelta(t, 1.011011, lg.GetEdgeIteratorState(sc2, 3).GetWeight(false), 1e-3)
}

func TestRoutingCHGraph_WeightExact(t *testing.T) {
	graph, speedEnc := newSpeedGraphDefault(t)
	graph.Edge(0, 1).SetDistance(1).SetDecimal(speedEnc, 60)
	graph.Edge(1, 2).SetDistance(1).SetDecimal(speedEnc, 60)
	graph.Freeze()

	chGraph, _, chBuilder := newNodeBasedCHGraph(t, graph, speedEnc, "ch")

	// 1.004+1.006 = 2.09999... we make sure this does not become 2.09 instead of 2.10 (due to truncation)
	x1 := 1.004
	x2 := 1.006
	chBuilder.AddShortcutNodeBased(0, 2, ch.ScFwdDir, x1+x2, 0, 1)
	sc := chGraph.GetEdgeIteratorState(2, 2)
	assert.InDelta(t, 2.01, sc.GetWeight(false), 1e-6)
}

func TestRoutingCHGraph_SimpleShortcutCreationAndTraversal(t *testing.T) {
	graph, speedEnc := newSpeedGraphDefault(t)
	graph.Edge(1, 3).SetDistance(10).SetDecimal(speedEnc, 60)
	graph.Edge(3, 4).SetDistance(10).SetDecimal(speedEnc, 60)
	graph.Freeze()

	chGraph, _, chBuilder := newNodeBasedCHGraph(t, graph, speedEnc, "p1")
	chBuilder.AddShortcutNodeBased(1, 4, ch.ScFwdDir, 3, util.NoEdge, util.NoEdge)

	// iteration should result in same nodes even if reusing the iterator
	exp := chGraph.CreateOutEdgeExplorer()
	assert.Equal(t, util.AsSet(3, 4), chNeighbors(exp.SetBaseNode(1)))
}

func TestRoutingCHGraph_SkippedEdgesWriteRead(t *testing.T) {
	graph, speedEnc := newSpeedGraphDefault(t)
	edge1 := graph.Edge(1, 3).SetDistance(10).SetDecimal(speedEnc, 60)
	edge2 := graph.Edge(3, 4).SetDistance(10).SetDecimal(speedEnc, 60)
	graph.Freeze()

	store := storage.CHStorageFromGraph(graph, "p1", false)
	chBuilder := storage.NewCHStorageBuilder(store)
	chBuilder.SetIdentityLevels()
	chBuilder.AddShortcutNodeBased(1, 4, ch.ScDirMask, 10, util.NoEdge, util.NoEdge)

	store.SetSkippedEdges(store.ToShortcutPointer(0), edge1.GetEdge(), edge2.GetEdge())
	assert.Equal(t, edge1.GetEdge(), store.GetSkippedEdge1(store.ToShortcutPointer(0)))
	assert.Equal(t, edge2.GetEdge(), store.GetSkippedEdge2(store.ToShortcutPointer(0)))
}

func TestRoutingCHGraph_SkippedEdges(t *testing.T) {
	graph, speedEnc := newSpeedGraphDefault(t)
	edge1 := graph.Edge(1, 3).SetDistance(10).SetDecimal(speedEnc, 60)
	edge2 := graph.Edge(3, 4).SetDistance(10).SetDecimal(speedEnc, 60)
	graph.Freeze()

	store := storage.CHStorageFromGraph(graph, "p1", false)
	chBuilder := storage.NewCHStorageBuilder(store)
	chBuilder.SetIdentityLevels()
	chBuilder.AddShortcutNodeBased(1, 4, ch.ScDirMask, 10, edge1.GetEdge(), edge2.GetEdge())
	assert.Equal(t, edge1.GetEdge(), store.GetSkippedEdge1(store.ToShortcutPointer(0)))
	assert.Equal(t, edge2.GetEdge(), store.GetSkippedEdge2(store.ToShortcutPointer(0)))
}

func TestRoutingCHGraph_EdgeBasedThrowsIfNotConfigured(t *testing.T) {
	graph, speedEnc := newSpeedGraphDefault(t)
	graph.Edge(0, 1).SetDistance(1).SetDecimal(speedEnc, 60)
	graph.Edge(1, 2).SetDistance(1).SetDecimal(speedEnc, 60)
	graph.Freeze()

	store := storage.CHStorageFromGraph(graph, "p1", false)
	chBuilder := storage.NewCHStorageBuilder(store)
	require.Panics(t, func() {
		chBuilder.AddShortcutEdgeBased(0, 2, ch.ScFwdDir, 10, 0, 1, 0, 2)
	})
}

func TestRoutingCHGraph_AddShortcutEdgeBased(t *testing.T) {
	// 0 -> 1 -> 2
	graph, speedEnc := newSpeedGraphDefault(t, withElevation)
	graph.Edge(0, 1).SetDistance(1).SetDecimal(speedEnc, 60)
	graph.Edge(1, 2).SetDistance(3).SetDecimal(speedEnc, 60)
	graph.Freeze()

	store := storage.CHStorageFromGraph(graph, "p1", true)
	chBuilder := storage.NewCHStorageBuilder(store)
	chBuilder.SetIdentityLevels()
	chBuilder.AddShortcutEdgeBased(0, 2, ch.ScFwdDir, 10, 0, 1, 0, 2)
	assert.Equal(t, 0, store.GetOrigEdgeKeyFirst(store.ToShortcutPointer(0)))
	assert.Equal(t, 2, store.GetOrigEdgeKeyLast(store.ToShortcutPointer(0)))
}

func TestRoutingCHGraph_OutOfBounds(t *testing.T) {
	graph, speedEnc := newSpeedGraphDefault(t, withElevation)
	graph.Freeze()

	w := weighting.NewSpeedWeighting(speedEnc)
	store := storage.CHStorageFromGraph(graph, "p1", false)
	lg := storage.NewRoutingCHGraph(graph, store, w)
	require.Panics(t, func() {
		lg.GetEdgeIteratorState(0, math.MinInt32)
	})
}

func TestRoutingCHGraph_GetEdgeIterator(t *testing.T) {
	graph, speedEnc := newSpeedGraphDefault(t, withElevation)
	graph.Edge(0, 1).SetDistance(1).SetDecimal(speedEnc, 60)
	graph.Edge(1, 2).SetDistance(1).SetDecimal(speedEnc, 60)
	graph.Freeze()

	w := weighting.NewSpeedWeighting(speedEnc)
	store := storage.CHStorageFromGraph(graph, "p1", true)
	chBuilder := storage.NewCHStorageBuilder(store)
	chBuilder.SetIdentityLevels()
	chBuilder.AddShortcutEdgeBased(0, 2, ch.ScFwdDir, 10, 0, 1, 0, 2)

	lg := storage.NewRoutingCHGraph(graph, store, w)

	sc02 := lg.GetEdgeIteratorState(2, 2)
	require.NotNil(t, sc02)
	assert.Equal(t, 0, sc02.GetBaseNode())
	assert.Equal(t, 2, sc02.GetAdjNode())
	assert.Equal(t, 2, sc02.GetEdge())
	assert.Equal(t, 0, sc02.GetSkippedEdge1())
	assert.Equal(t, 1, sc02.GetSkippedEdge2())
	assert.Equal(t, 0, sc02.GetOrigEdgeKeyFirst())
	assert.Equal(t, 2, sc02.GetOrigEdgeKeyLast())

	sc20 := lg.GetEdgeIteratorState(2, 0)
	require.NotNil(t, sc20)
	assert.Equal(t, 2, sc20.GetBaseNode())
	assert.Equal(t, 0, sc20.GetAdjNode())
	assert.Equal(t, 2, sc20.GetEdge())
	// note these are not stateful! i.e. even though we are looking at the edge 2->0 the first skipped/orig edge
	// is still edge 0 and the second skipped/last orig edge is edge 1
	assert.Equal(t, 0, sc20.GetSkippedEdge1())
	assert.Equal(t, 1, sc20.GetSkippedEdge2())
	assert.Equal(t, 0, sc20.GetOrigEdgeKeyFirst())
	assert.Equal(t, 2, sc20.GetOrigEdgeKeyLast())
}

func TestRoutingCHGraph_FwdLoopShortcut(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	turnCostEnc := ev.TurnCostCreate("car", 1)
	em := routingutil.Start().Add(speedEnc).AddTurnCostEncodedValue(turnCostEnc).Build()
	graph := storage.NewBaseGraphBuilder(em.BytesForFlags).
		SetWithTurnCosts(true).
		CreateGraph()

	// 0-1
	//  \|
	//   2
	graph.Edge(0, 1).SetDistance(100).SetDecimalBothDir(speedEnc, 60, 0)
	graph.Edge(1, 2).SetDistance(200).SetDecimalBothDir(speedEnc, 60, 0)
	graph.Edge(2, 0).SetDistance(300).SetDecimalBothDir(speedEnc, 60, 0)
	graph.Freeze()

	// add loop shortcut in 'fwd' direction
	w := weighting.NewSpeedWeightingWithTurnCosts(speedEnc, turnCostEnc, graph.GetTurnCostStorage(), graph.GetNodeAccess(), 50)
	store := storage.CHStorageFromGraph(graph, "profile", true)
	chBuilder := storage.NewCHStorageBuilder(store)
	chBuilder.SetIdentityLevels()
	chBuilder.AddShortcutEdgeBased(0, 0, ch.ScFwdDir, 5, 0, 2, 0, 5)
	chGraph := storage.NewRoutingCHGraph(graph, store, w)

	outEdges := chEdgeIDs(chGraph.CreateOutEdgeExplorer().SetBaseNode(0))
	inEdges := chEdgeIDs(chGraph.CreateInEdgeExplorer().SetBaseNode(0))

	// the loop should be accepted by in- and outExplorers
	assert.Equal(t, util.AsSet(0, 3), outEdges, "Wrong outgoing edges")
	assert.Equal(t, util.AsSet(2, 3), inEdges, "Wrong incoming edges")
}
