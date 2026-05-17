package util_test

import (
	"math"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// TestRandomGraph_SmokeFiftyNodes builds a 50-node graph at meanDegree=3
// and asserts the edge count matches Java's int(0.5 * meanDegree *
// numNodes) formula exactly, since RandomGraph adds one edge per loop
// iteration until it reaches that target.
func TestRandomGraph_SmokeFiftyNodes(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	em := routingutil.Start().Add(speedEnc).Build()
	g := storage.NewBaseGraphBuilder(em.BytesForFlags).CreateGraph()

	const (
		numNodes   = 50
		meanDegree = 3.0
	)
	rnd := rand.New(rand.NewSource(42))
	util.RandomGraph(g.GetNodeAccess(), g.Edge, rnd, numNodes, meanDegree, true, speedEnc, nil, 0.9, 0.8)

	assert.Equal(t, numNodes, g.GetNodes())
	expectedEdges := int(0.5 * meanDegree * float64(numNodes))
	assert.Equal(t, expectedEdges, g.GetEdges(), "edge count must equal Java's int(0.5*meanDegree*numNodes)")

	// Sanity: every node should be inside the (49.4..49.41, 9.7..9.71) bbox.
	na := g.GetNodeAccess()
	for i := 0; i < numNodes; i++ {
		assert.GreaterOrEqual(t, na.GetLat(i), 49.4)
		assert.Less(t, na.GetLat(i), 49.41)
		assert.GreaterOrEqual(t, na.GetLon(i), 9.7)
		assert.Less(t, na.GetLon(i), 9.71)
	}
}

// TestRandomGraph_DeterministicForSeed runs the helper twice with the
// same seed and asserts the resulting graphs are identical (same edge
// count, same per-edge endpoints and distances).
func TestRandomGraph_DeterministicForSeed(t *testing.T) {
	speedEnc1 := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	em1 := routingutil.Start().Add(speedEnc1).Build()
	g1 := storage.NewBaseGraphBuilder(em1.BytesForFlags).CreateGraph()
	util.RandomGraph(g1.GetNodeAccess(), g1.Edge, rand.New(rand.NewSource(7)), 20, 3, true, speedEnc1, nil, 0.5, 0.5)

	speedEnc2 := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	em2 := routingutil.Start().Add(speedEnc2).Build()
	g2 := storage.NewBaseGraphBuilder(em2.BytesForFlags).CreateGraph()
	util.RandomGraph(g2.GetNodeAccess(), g2.Edge, rand.New(rand.NewSource(7)), 20, 3, true, speedEnc2, nil, 0.5, 0.5)

	assert.Equal(t, g1.GetEdges(), g2.GetEdges())
	all1 := g1.GetAllEdges()
	all2 := g2.GetAllEdges()
	for all1.Next() {
		assert.True(t, all2.Next())
		assert.Equal(t, all1.GetBaseNode(), all2.GetBaseNode())
		assert.Equal(t, all1.GetAdjNode(), all2.GetAdjNode())
		assert.InDelta(t, all1.GetDistance(), all2.GetDistance(), 1e-9)
	}
	assert.False(t, all2.Next())
}

// TestAddRandomTurnCosts_RoundTrip builds a small graph, applies random
// turn costs with a fixed seed, and verifies that:
//  1. At least one turn cost is written (sanity check on the helper).
//  2. Each value read back from TurnCostStorage matches the value passed
//     to the setter closure, up to the encoded-value quantization
//     (TurnCost uses factor=1 so cost is quantized to integer steps).
//  3. Infinite costs (turn restrictions) survive the round-trip exactly.
func TestAddRandomTurnCosts_RoundTrip(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	turnCostEnc := ev.TurnCostCreate("car", 10)
	em := routingutil.Start().Add(speedEnc).AddTurnCostEncodedValue(turnCostEnc).Build()
	g := storage.NewBaseGraphBuilder(em.BytesForFlags).SetWithTurnCosts(true).CreateGraph()

	util.RandomGraph(g.GetNodeAccess(), g.Edge, rand.New(rand.NewSource(1)), 20, 3, true, speedEnc, nil, 0.9, 0.8)
	g.Freeze()

	tcs := g.GetTurnCostStorage()
	type tcKey struct{ from, via, to int }
	written := make(map[tcKey]float64)
	rnd := rand.New(rand.NewSource(1))
	util.AddRandomTurnCosts(g.GetNodes(), rnd, g.CreateEdgeExplorer(routingutil.AllEdges), g.CreateEdgeExplorer(routingutil.AllEdges), turnCostEnc, 10,
		func(enc ev.DecimalEncodedValue, fromEdge, viaNode, toEdge int, cost float64) {
			written[tcKey{fromEdge, viaNode, toEdge}] = cost
			tcs.SetDecimal(g.GetNodeAccess(), enc, fromEdge, viaNode, toEdge, cost)
		})

	assert.NotEmpty(t, written, "AddRandomTurnCosts produced no turn costs at all")
	const quantizationStep = 1.0 // TurnCost encoded value uses factor=1
	for k, expected := range written {
		got := tcs.GetDecimal(g.GetNodeAccess(), turnCostEnc, k.from, k.via, k.to)
		if math.IsInf(expected, 1) {
			assert.True(t, math.IsInf(got, 1), "infinite cost lost on round-trip at %+v: got %v", k, got)
			continue
		}
		assert.InDelta(t, expected, got, quantizationStep, "turn cost mismatch at %+v", k)
	}
}
