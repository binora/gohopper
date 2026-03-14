package storage_test

import (
	"math"
	"testing"

	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/storage"
	"gohopper/core/util"

	"github.com/stretchr/testify/assert"
)

// --- Turn cost test helpers ---

// turnCostTestGraph holds the shared state for turn cost tests.
type turnCostTestGraph struct {
	graph       *storage.BaseGraph
	tc          *storage.TurnCostStorage
	na          storage.NodeAccess
	carTCEnc    ev.DecimalEncodedValue
	bikeTCEnc   ev.DecimalEncodedValue
}

// newTurnCostTestGraph creates a BaseGraph with car access/speed plus car and bike
// turn cost encoders, initializes the standard TC graph topology, and returns
// the assembled test state.
//
// 0---1
// |   /
// 2--3
// |
// 4
func newTurnCostTestGraph(t *testing.T) *turnCostTestGraph {
	t.Helper()
	accessEnc := ev.NewSimpleBooleanEncodedValueDir("car_access", true)
	speedEnc := ev.NewDecimalEncodedValueImpl("car_speed", 5, 5, false)
	carTCEnc := ev.TurnCostCreate("car", 3)
	bikeTCEnc := ev.TurnCostCreate("bike", 3)
	em := routingutil.Start().
		Add(accessEnc).Add(speedEnc).
		AddTurnCostEncodedValue(carTCEnc).AddTurnCostEncodedValue(bikeTCEnc).Build()
	g := storage.NewBaseGraphBuilder(em.BytesForFlags).SetWithTurnCosts(true).CreateGraph()
	initTCGraph(g, accessEnc, speedEnc)
	return &turnCostTestGraph{
		graph:     g,
		tc:        g.GetTurnCostStorage(),
		na:        g.GetNodeAccess(),
		carTCEnc:  carTCEnc,
		bikeTCEnc: bikeTCEnc,
	}
}

func getEdge(g *storage.BaseGraph, from, to int) util.EdgeIteratorState {
	iter := g.CreateEdgeExplorer(routingutil.AllEdges).SetBaseNode(from)
	for iter.Next() {
		if iter.GetAdjNode() == to {
			return iter
		}
	}
	panic("edge not found")
}

func initTCGraph(g *storage.BaseGraph, accessEnc ev.BooleanEncodedValue, speedEnc ev.DecimalEncodedValue) {
	util.SetSpeeds(60, 60, accessEnc, speedEnc,
		g.Edge(0, 1).SetDistance(3),
		g.Edge(0, 2).SetDistance(1),
		g.Edge(1, 3).SetDistance(1),
		g.Edge(2, 3).SetDistance(1),
		g.Edge(2, 4).SetDistance(1),
	)
}

// tcKey identifies a turn cost entry by its edge pair and per-profile integer costs.
type tcKey struct {
	from, via, to, car, bike int
}

// costToInt converts a turn cost float to an int, mapping +Inf to MaxInt32.
func costToInt(cost float64) int {
	if math.IsInf(cost, 1) {
		return math.MaxInt32
	}
	return int(cost)
}

// collectAllTurnCosts iterates all turn costs and returns them as a set of tcKey.
func collectAllTurnCosts(tc *storage.TurnCostStorage, na storage.NodeAccess, nodes int, carEnc, bikeEnc ev.DecimalEncodedValue) map[tcKey]bool {
	result := make(map[tcKey]bool)
	iter := tc.GetAllTurnCosts(na, nodes)
	for iter.Next() {
		key := tcKey{
			from: iter.GetFromEdge(),
			via:  iter.GetViaNode(),
			to:   iter.GetToEdge(),
			car:  costToInt(iter.GetCost(carEnc)),
			bike: costToInt(iter.GetCost(bikeEnc)),
		}
		result[key] = true
	}
	return result
}

// --- Tests ---

func TestTurnCostStorage_MultipleTurnCosts(t *testing.T) {
	tg := newTurnCostTestGraph(t)
	tc, na := tg.tc, tg.na
	carEnc, bikeEnc := tg.carTCEnc, tg.bikeTCEnc

	edge42 := getEdge(tg.graph, 4, 2).GetEdge()
	edge23 := getEdge(tg.graph, 2, 3).GetEdge()
	edge31 := getEdge(tg.graph, 3, 1).GetEdge()
	edge10 := getEdge(tg.graph, 1, 0).GetEdge()
	edge02 := getEdge(tg.graph, 0, 2).GetEdge()
	edge24 := getEdge(tg.graph, 2, 4).GetEdge()

	tc.SetDecimal(na, carEnc, edge42, 2, edge23, math.Inf(1))
	tc.SetDecimal(na, bikeEnc, edge42, 2, edge23, math.Inf(1))
	tc.SetDecimal(na, carEnc, edge23, 3, edge31, math.Inf(1))
	tc.SetDecimal(na, bikeEnc, edge23, 3, edge31, 2.0)
	tc.SetDecimal(na, carEnc, edge31, 1, edge10, 2.0)
	tc.SetDecimal(na, bikeEnc, edge31, 1, edge10, math.Inf(1))
	tc.SetDecimal(na, bikeEnc, edge02, 2, edge24, math.Inf(1))

	// count check: total == sum of per-node counts
	totalCount := tc.Count()
	sumPerNode := 0
	for node := 0; node < tg.graph.GetNodes(); node++ {
		sumPerNode += tc.GetTurnCostsCount(na, node)
	}
	assert.Equal(t, totalCount, sumPerNode)

	assert.Equal(t, math.Inf(1), tc.GetDecimal(na, carEnc, edge42, 2, edge23))
	assert.Equal(t, math.Inf(1), tc.GetDecimal(na, bikeEnc, edge42, 2, edge23))

	assert.Equal(t, math.Inf(1), tc.GetDecimal(na, carEnc, edge23, 3, edge31))
	assert.Equal(t, 2.0, tc.GetDecimal(na, bikeEnc, edge23, 3, edge31))

	assert.Equal(t, 2.0, tc.GetDecimal(na, carEnc, edge31, 1, edge10))
	assert.Equal(t, math.Inf(1), tc.GetDecimal(na, bikeEnc, edge31, 1, edge10))

	assert.Equal(t, 0.0, tc.GetDecimal(na, carEnc, edge02, 2, edge24))
	assert.Equal(t, math.Inf(1), tc.GetDecimal(na, bikeEnc, edge02, 2, edge24))

	tc.SetDecimal(na, carEnc, edge02, 2, edge23, math.Inf(1))
	tc.SetDecimal(na, bikeEnc, edge02, 2, edge23, math.Inf(1))
	assert.Equal(t, math.Inf(1), tc.GetDecimal(na, carEnc, edge02, 2, edge23))
	assert.Equal(t, math.Inf(1), tc.GetDecimal(na, bikeEnc, edge02, 2, edge23))

	// iterate all turn costs and compare with expected set
	turnCosts := collectAllTurnCosts(tc, na, tg.graph.GetNodes(), carEnc, bikeEnc)
	expected := map[tcKey]bool{
		{edge31, 1, edge10, 2, math.MaxInt32}:              true,
		{edge42, 2, edge23, math.MaxInt32, math.MaxInt32}:  true,
		{edge02, 2, edge24, 0, math.MaxInt32}:              true,
		{edge02, 2, edge23, math.MaxInt32, math.MaxInt32}:  true,
		{edge23, 3, edge31, math.MaxInt32, 2}:              true,
	}
	assert.Equal(t, expected, turnCosts)
}

func TestTurnCostStorage_MergeFlagsBeforeAdding(t *testing.T) {
	tg := newTurnCostTestGraph(t)
	tc, na := tg.tc, tg.na
	carEnc, bikeEnc := tg.carTCEnc, tg.bikeTCEnc

	edge23 := getEdge(tg.graph, 2, 3).GetEdge()
	edge02 := getEdge(tg.graph, 0, 2).GetEdge()

	tc.SetDecimal(na, carEnc, edge02, 2, edge23, math.Inf(1))
	tc.SetDecimal(na, bikeEnc, edge02, 2, edge23, math.Inf(1))
	assert.Equal(t, math.Inf(1), tc.GetDecimal(na, carEnc, edge02, 2, edge23))
	assert.Equal(t, math.Inf(1), tc.GetDecimal(na, bikeEnc, edge02, 2, edge23))

	turnCosts := collectAllTurnCosts(tc, na, tg.graph.GetNodes(), carEnc, bikeEnc)
	expected := map[tcKey]bool{
		{edge02, 2, edge23, math.MaxInt32, math.MaxInt32}: true,
	}
	assert.Equal(t, expected, turnCosts)
}

func TestTurnCostStorage_SetMultipleTimes(t *testing.T) {
	tg := newTurnCostTestGraph(t)
	tc, na := tg.tc, tg.na
	carEnc := tg.carTCEnc

	edge32 := getEdge(tg.graph, 3, 2).GetEdge()
	edge20 := getEdge(tg.graph, 2, 0).GetEdge()

	assert.Equal(t, 0.0, tc.GetDecimal(na, carEnc, edge32, 2, edge20))
	tc.SetDecimal(na, carEnc, edge32, 2, edge20, math.Inf(1))
	assert.Equal(t, math.Inf(1), tc.GetDecimal(na, carEnc, edge32, 2, edge20))
	tc.SetDecimal(na, carEnc, edge32, 2, edge20, 0)
	tc.SetDecimal(na, carEnc, edge32, 2, edge20, math.Inf(1))
	tc.SetDecimal(na, carEnc, edge32, 2, edge20, 0)
	assert.Equal(t, 0.0, tc.GetDecimal(na, carEnc, edge32, 2, edge20))
}

func TestTurnCostStorage_IterateEmptyStore(t *testing.T) {
	tg := newTurnCostTestGraph(t)
	iterator := tg.tc.GetAllTurnCosts(tg.na, tg.graph.GetNodes())
	assert.False(t, iterator.Next())
}
