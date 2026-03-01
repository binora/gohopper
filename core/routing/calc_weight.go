package routing

import (
	"gohopper/core/routing/weighting"
	"gohopper/core/util"
)

// CalcWeightWithTurnWeight calculates the weight of a given edge like
// Weighting.CalcEdgeWeight and adds the transition cost (turn weight)
// associated with transitioning from/to the edge with prevOrNextEdgeID.
//
// If reverse is false, prevOrNextEdgeID must be the previous edge ID.
// If reverse is true, prevOrNextEdgeID must be the next edge ID
// (in the direction from start to end).
func CalcWeightWithTurnWeight(w weighting.Weighting, edgeState util.EdgeIteratorState, reverse bool, prevOrNextEdgeID int) float64 {
	edgeWeight := w.CalcEdgeWeight(edgeState, reverse)
	if !util.EdgeIsValid(prevOrNextEdgeID) {
		return edgeWeight
	}
	var turnWeight float64
	if reverse {
		turnWeight = w.CalcTurnWeight(edgeState.GetEdge(), edgeState.GetBaseNode(), prevOrNextEdgeID)
	} else {
		turnWeight = w.CalcTurnWeight(prevOrNextEdgeID, edgeState.GetBaseNode(), edgeState.GetEdge())
	}
	return edgeWeight + turnWeight
}
