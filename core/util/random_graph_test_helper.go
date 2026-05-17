package util

import (
	"math"
	"math/rand"

	"gohopper/core/routing/ev"
)

// RandomGraphEdgeFactory mirrors the BaseGraph.Edge constructor. RandomGraph
// takes this as a function (rather than typing against *storage.BaseGraph
// directly) because core/storage already imports core/util — taking a
// typed graph argument would create an import cycle.
type RandomGraphEdgeFactory func(nodeA, nodeB int) EdgeIteratorState

// RandomGraphNodeAccess is the NodeAccess surface that RandomGraph needs.
// *storage.BaseGraph's NodeAccess satisfies it directly.
type RandomGraphNodeAccess interface {
	SetNode(nodeID int, lat, lon, ele float64)
	GetLat(nodeID int) float64
	GetLon(nodeID int) float64
}

// RandomGraphTurnCostSetter is the closure AddRandomTurnCosts uses to write
// a single turn cost (via TurnCostStorage.SetDecimal in production).
type RandomGraphTurnCostSetter func(turnCostEnc ev.DecimalEncodedValue, fromEdge, viaNode, toEdge int, cost float64)

// RandomGraph mirrors com.graphhopper.util.GHUtility#buildRandomGraph.
//
// It seeds numNodes around the (49.4..49.41, 9.7..9.71) bounding box and
// keeps adding edges until roughly 0.5 * meanDegree * numNodes have been
// created. Duplicate edges between the same node pair are allowed. When
// speedEnc is non-nil and speed is nil, random forward/reverse speeds in
// [10, 120) are assigned; when speed is non-nil that explicit value is
// used for both directions. pBothDir controls how often an edge is
// bidirectional; pRandomDistanceOffset controls how often a tiny offset is
// added to the base distance.
//
// Parity note: Java's java.util.Random and Go's math/rand use different
// PRNGs, so identical seeds will NOT reproduce identical graphs across
// runtimes. The structural distribution (counts and value ranges) matches.
func RandomGraph(na RandomGraphNodeAccess, edge RandomGraphEdgeFactory, rnd *rand.Rand, numNodes int, meanDegree float64, allowZeroDistance bool, speedEnc ev.DecimalEncodedValue, speed *float64, pBothDir, pRandomDistanceOffset float64) {
	if numNodes < 2 || meanDegree < 1 {
		panic("numNodes must be >= 2, meanDegree >= 1")
	}
	for i := 0; i < numNodes; i++ {
		lat := 49.4 + rnd.Float64()*0.01
		lon := 9.7 + rnd.Float64()*0.01
		na.SetNode(i, lat, lon, math.NaN())
	}
	totalNumEdges := int(0.5 * meanDegree * float64(numNodes))
	for numEdges := 0; numEdges < totalNumEdges; numEdges++ {
		var from, to int
		for {
			from = rnd.Intn(numNodes)
			to = rnd.Intn(numNodes)
			if from != to {
				break
			}
		}
		distance := DistPlane.CalcDist(na.GetLat(from), na.GetLon(from), na.GetLat(to), na.GetLon(to))
		if !allowZeroDistance {
			distance = math.Max(0.001, distance)
		}
		// add a small random offset, but also allow duplicate edges with the same weight
		if rnd.Float64() < pRandomDistanceOffset {
			distance += rnd.Float64() * distance * 0.01
		}
		// bidirectional edges raise effective mean degree above the parameter
		bothDirections := rnd.Float64() < pBothDir
		e := edge(from, to).SetDistance(distance)
		fwdSpeed := 10 + rnd.Float64()*110
		bwdSpeed := 10 + rnd.Float64()*110
		if speed != nil {
			fwdSpeed = *speed
			bwdSpeed = *speed
		}
		if speedEnc == nil {
			continue
		}
		e.SetDecimal(speedEnc, fwdSpeed)
		if speedEnc.IsStoreTwoDirections() {
			reverse := bwdSpeed
			if !bothDirections {
				reverse = 0
			}
			e.SetReverseDecimal(speedEnc, reverse)
		}
	}
}

// AddRandomTurnCosts mirrors com.graphhopper.util.GHUtility#addRandomTurnCosts.
//
// For every node, with probability pNodeHasTurnCosts it walks each
// (in, out) edge pair; with probability pEdgePairHasTurnCosts a cost is
// assigned — either a finite value in [0, maxTurnCost) or, with
// probability pCostIsRestriction, an infinite cost (turn restriction).
// u-turns (in == out) are skipped, matching Java.
//
// inExplorer/outExplorer let the caller fold Java's accessEnc argument
// into direction-aware filters; passing accept-all explorers matches
// Java's "accessEnc == null" branch.
func AddRandomTurnCosts(numNodes int, rnd *rand.Rand, inExplorer, outExplorer EdgeExplorer, turnCostEnc ev.DecimalEncodedValue, maxTurnCost int, set RandomGraphTurnCostSetter) {
	const (
		pNodeHasTurnCosts     = 0.3
		pEdgePairHasTurnCosts = 0.6
		pCostIsRestriction    = 0.1
	)
	for node := 0; node < numNodes; node++ {
		if rnd.Float64() >= pNodeHasTurnCosts {
			continue
		}
		inIter := inExplorer.SetBaseNode(node)
		for inIter.Next() {
			outIter := outExplorer.SetBaseNode(node)
			for outIter.Next() {
				if inIter.GetEdge() == outIter.GetEdge() {
					continue // leave u-turns as they are
				}
				if rnd.Float64() >= pEdgePairHasTurnCosts {
					continue
				}
				var cost float64
				if rnd.Float64() < pCostIsRestriction {
					cost = math.Inf(1)
				} else {
					cost = rnd.Float64() * float64(maxTurnCost)
				}
				set(turnCostEnc, inIter.GetEdge(), node, outIter.GetEdge(), cost)
			}
		}
	}
}
