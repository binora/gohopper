package routing

import (
	"container/heap"
	"fmt"
	"math"

	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// AStar implements the A* shortest-path algorithm.
type AStar struct {
	AbstractRoutingAlgorithm
	fromMap      map[int]*SPTEntry
	fromHeap     sptEntryHeap
	currEdge     *SPTEntry
	to           int
	weightApprox weighting.WeightApproximator
	fromOutEdge  int
	toInEdge     int
}

func NewAStar(graph storage.Graph, w weighting.Weighting, tMode routingutil.TraversalMode) *AStar {
	size := min(max(graph.GetNodes()/10, 200), 2000)
	a := &AStar{
		AbstractRoutingAlgorithm: NewAbstractRoutingAlgorithm(graph, w, tMode),
		to:                       -1,
		fromOutEdge:              util.AnyEdge,
		toInEdge:                 util.AnyEdge,
		fromHeap:                 make(sptEntryHeap, 0, size),
		fromMap:                  make(map[int]*SPTEntry, size),
	}
	defaultApprox := weighting.NewBeelineWeightApproximator(a.NodeAccess, w)
	defaultApprox.SetDistanceCalc(util.DistPlane)
	a.SetApproximation(defaultApprox)
	return a
}

func (a *AStar) SetApproximation(approx weighting.WeightApproximator) *AStar {
	a.weightApprox = approx
	return a
}

func (a *AStar) CalcPath(from, to int) *Path {
	return a.CalcPathEdgeToEdge(from, to, util.AnyEdge, util.AnyEdge)
}

func (a *AStar) CalcPathEdgeToEdge(from, to, fromOutEdge, toInEdge int) *Path {
	if (fromOutEdge != util.AnyEdge || toInEdge != util.AnyEdge) && !a.TraversalMode.IsEdgeBased() {
		panic("Restricting the start/target edges is only possible for edge-based graph traversal")
	}
	a.fromOutEdge = fromOutEdge
	a.toInEdge = toInEdge
	a.CheckAlreadyRun()
	a.SetupFinishTime()
	a.to = to

	if fromOutEdge == util.NoEdge || toInEdge == util.NoEdge {
		return a.extractPath()
	}

	a.weightApprox.SetTo(to)
	weightToGoal := a.weightApprox.Approximate(from)
	if math.IsInf(weightToGoal, 1) {
		return a.extractPath()
	}

	startEntry := NewSPTEntryWithHeuristic(util.NoEdge, from, weightToGoal, 0, nil)
	heap.Push(&a.fromHeap, startEntry)
	if !a.TraversalMode.IsEdgeBased() {
		a.fromMap[from] = startEntry
	}
	a.runAlgo()
	return a.extractPath()
}

func (a *AStar) CalcPaths(from, to int) []*Path {
	return DefaultCalcPaths(a, from, to)
}

func (a *AStar) runAlgo() {
	for a.fromHeap.Len() > 0 {
		a.currEdge = heap.Pop(&a.fromHeap).(*SPTEntry)
		if a.currEdge.Deleted {
			continue
		}
		a.VisitedNodes++
		if a.IsMaxVisitedNodesExceeded() || a.finished() || a.IsTimeoutExceeded() {
			break
		}

		currNode := a.currEdge.AdjNode
		iter := a.EdgeExplorer.SetBaseNode(currNode)
		for iter.Next() {
			if !a.Accept(iter, a.currEdge.Edge) {
				continue
			}
			if a.currEdge.Edge == util.NoEdge && a.fromOutEdge != util.AnyEdge && iter.GetEdge() != a.fromOutEdge {
				continue
			}

			tmpWeight := CalcWeightWithTurnWeight(a.Weighting, iter, false, a.currEdge.Edge) + a.currEdge.WeightOfVisitedPath
			if math.IsInf(tmpWeight, 1) {
				continue
			}

			traversalID := a.TraversalMode.CreateTraversalID(iter, false)
			ase := a.fromMap[traversalID]
			if ase == nil || ase.WeightOfVisitedPath > tmpWeight {
				neighborNode := iter.GetAdjNode()
				currWeightToGoal := a.weightApprox.Approximate(neighborNode)
				if math.IsInf(currWeightToGoal, 1) {
					continue
				}
				estimationFullWeight := tmpWeight + currWeightToGoal
				if ase != nil {
					ase.Deleted = true
				}
				ase = NewSPTEntryWithHeuristic(iter.GetEdge(), neighborNode, estimationFullWeight, tmpWeight, a.currEdge)
				a.fromMap[traversalID] = ase
				heap.Push(&a.fromHeap, ase)
			}
		}
	}
}

func (a *AStar) finished() bool {
	return a.currEdge.AdjNode == a.to &&
		(a.toInEdge == util.AnyEdge || a.currEdge.Edge == a.toInEdge) &&
		(a.fromOutEdge == util.AnyEdge || a.currEdge.Edge != util.NoEdge)
}

func (a *AStar) extractPath() *Path {
	if a.currEdge == nil || !a.finished() {
		return a.CreateEmptyPath()
	}
	path := ExtractPath(a.Graph, a.Weighting, a.currEdge)
	path.SetWeight(a.currEdge.WeightOfVisitedPath)
	return path
}

func (a *AStar) GetName() string {
	return fmt.Sprintf("%s|%s", AlgoAStar, a.weightApprox)
}
