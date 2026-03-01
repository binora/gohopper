package weighting

import (
	"math"

	routingutil "gohopper/core/routing/util"
	"gohopper/core/storage"
	"gohopper/core/util"
)

var _ Weighting = (*QueryGraphWeighting)(nil)

type QueryGraphWeighting struct {
	graph              *storage.BaseGraph
	weighting          Weighting
	firstVirtualNodeID int
	firstVirtualEdgeID int
	closestEdges       []int
}

func NewQueryGraphWeighting(graph *storage.BaseGraph, weighting Weighting, closestEdges []int) *QueryGraphWeighting {
	return &QueryGraphWeighting{
		graph:              graph,
		weighting:          weighting,
		firstVirtualNodeID: graph.GetNodes(),
		firstVirtualEdgeID: graph.GetEdges(),
		closestEdges:       closestEdges,
	}
}

func (w *QueryGraphWeighting) CalcMinWeightPerDistance() float64 {
	return w.weighting.CalcMinWeightPerDistance()
}

func (w *QueryGraphWeighting) CalcEdgeWeight(edgeState util.EdgeIteratorState, reverse bool) float64 {
	return w.weighting.CalcEdgeWeight(edgeState, reverse)
}

func (w *QueryGraphWeighting) CalcTurnWeight(inEdge, viaNode, outEdge int) float64 {
	if !util.EdgeIsValid(inEdge) || !util.EdgeIsValid(outEdge) {
		return 0
	}
	if w.isVirtualNode(viaNode) {
		if inEdge == outEdge {
			return math.Inf(1)
		}
		return 0
	}
	return w.getMinWeightAndOriginalEdges(inEdge, viaNode, outEdge).minTurnWeight
}

type turnCostResult struct {
	origInEdge    int
	origOutEdge   int
	minTurnWeight float64
}

func (w *QueryGraphWeighting) getMinWeightAndOriginalEdges(inEdge, viaNode, outEdge int) turnCostResult {
	r := turnCostResult{
		origInEdge:    -1,
		origOutEdge:   -1,
		minTurnWeight: math.Inf(1),
	}
	if w.isVirtualEdge(inEdge) && w.isVirtualEdge(outEdge) {
		innerExplorer := w.graph.CreateEdgeExplorer(routingutil.AllEdges)
		w.graph.ForEdgeAndCopiesOfEdge(w.graph.CreateEdgeExplorer(routingutil.AllEdges), viaNode, w.getOriginalEdge(inEdge), func(p int) {
			w.graph.ForEdgeAndCopiesOfEdge(innerExplorer, viaNode, w.getOriginalEdge(outEdge), func(q int) {
				tw := w.weighting.CalcTurnWeight(p, viaNode, q)
				if tw < r.minTurnWeight {
					r.origInEdge = p
					r.origOutEdge = q
					r.minTurnWeight = tw
				}
			})
		})
	} else if w.isVirtualEdge(inEdge) {
		w.graph.ForEdgeAndCopiesOfEdge(w.graph.CreateEdgeExplorer(routingutil.AllEdges), viaNode, w.getOriginalEdge(inEdge), func(e int) {
			tw := w.weighting.CalcTurnWeight(e, viaNode, outEdge)
			if tw < r.minTurnWeight {
				r.origInEdge = e
				r.origOutEdge = outEdge
				r.minTurnWeight = tw
			}
		})
	} else if w.isVirtualEdge(outEdge) {
		w.graph.ForEdgeAndCopiesOfEdge(w.graph.CreateEdgeExplorer(routingutil.AllEdges), viaNode, w.getOriginalEdge(outEdge), func(e int) {
			tw := w.weighting.CalcTurnWeight(inEdge, viaNode, e)
			if tw < r.minTurnWeight {
				r.origInEdge = inEdge
				r.origOutEdge = e
				r.minTurnWeight = tw
			}
		})
	} else {
		r.origInEdge = inEdge
		r.origOutEdge = outEdge
		r.minTurnWeight = w.weighting.CalcTurnWeight(inEdge, viaNode, outEdge)
	}
	return r
}

func (w *QueryGraphWeighting) CalcEdgeMillis(edgeState util.EdgeIteratorState, reverse bool) int64 {
	return w.weighting.CalcEdgeMillis(edgeState, reverse)
}

func (w *QueryGraphWeighting) CalcTurnMillis(inEdge, viaNode, outEdge int) int64 {
	if w.isVirtualNode(viaNode) {
		return 0
	}
	r := w.getMinWeightAndOriginalEdges(inEdge, viaNode, outEdge)
	return w.weighting.CalcTurnMillis(r.origInEdge, viaNode, r.origOutEdge)
}

func (w *QueryGraphWeighting) HasTurnCosts() bool {
	return w.weighting.HasTurnCosts()
}

func (w *QueryGraphWeighting) GetName() string {
	return w.weighting.GetName()
}

func (w *QueryGraphWeighting) String() string {
	return w.GetName()
}

func (w *QueryGraphWeighting) getOriginalEdge(edge int) int {
	return w.closestEdges[(edge-w.firstVirtualEdgeID)/2]
}

func (w *QueryGraphWeighting) isVirtualNode(node int) bool {
	return node >= w.firstVirtualNodeID
}

func (w *QueryGraphWeighting) isVirtualEdge(edge int) bool {
	return edge >= w.firstVirtualEdgeID
}
