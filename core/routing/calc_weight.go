package routing

import (
	"gohopper/core/routing/weighting"
	"gohopper/core/util"
)

// CalcWeightWithTurnWeight calculates edge weight plus turn cost for the
// transition from/to the edge identified by prevOrNextEdgeID.
func CalcWeightWithTurnWeight(w weighting.Weighting, edgeState util.EdgeIteratorState, reverse bool, prevOrNextEdgeID int) float64 {
	edgeWeight := w.CalcEdgeWeight(edgeState, reverse)
	if !util.EdgeIsValid(prevOrNextEdgeID) {
		return edgeWeight
	}
	if reverse {
		return edgeWeight + w.CalcTurnWeight(edgeState.GetEdge(), edgeState.GetBaseNode(), prevOrNextEdgeID)
	}
	return edgeWeight + w.CalcTurnWeight(prevOrNextEdgeID, edgeState.GetBaseNode(), edgeState.GetEdge())
}
