package weighting

import "gohopper/core/util"

var _ Weighting = (*AvoidEdgesWeighting)(nil)

// AvoidEdgesWeighting increases the weight for a certain set of edges by a
// given factor and thus makes them less likely to be part of a shortest path.
type AvoidEdgesWeighting struct {
	AbstractAdjustedWeighting
	AvoidedEdges      map[int]struct{}
	EdgePenaltyFactor float64
}

func NewAvoidEdgesWeighting(superWeighting Weighting) *AvoidEdgesWeighting {
	return &AvoidEdgesWeighting{
		AbstractAdjustedWeighting: NewAbstractAdjustedWeighting(superWeighting),
		AvoidedEdges:              make(map[int]struct{}),
		EdgePenaltyFactor:         5.0,
	}
}

func (w *AvoidEdgesWeighting) SetEdgePenaltyFactor(factor float64) *AvoidEdgesWeighting {
	w.EdgePenaltyFactor = factor
	return w
}

func (w *AvoidEdgesWeighting) SetAvoidedEdges(edges map[int]struct{}) *AvoidEdgesWeighting {
	w.AvoidedEdges = edges
	return w
}

func (w *AvoidEdgesWeighting) CalcEdgeWeight(edgeState util.EdgeIteratorState, reverse bool) float64 {
	weight := w.SuperWeighting.CalcEdgeWeight(edgeState, reverse)
	if _, ok := w.AvoidedEdges[edgeState.GetEdge()]; ok {
		return weight * w.EdgePenaltyFactor
	}
	return weight
}

func (w *AvoidEdgesWeighting) GetName() string {
	return "avoid_edges"
}
