package routing

import (
	"math"

	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/util"
)

func NewFiniteWeightFilter(w weighting.Weighting) routingutil.EdgeFilter {
	return func(edgeState util.EdgeIteratorState) bool {
		return isFinite(w.CalcEdgeWeight(edgeState, false)) ||
			isFinite(w.CalcEdgeWeight(edgeState, true))
	}
}

func isFinite(v float64) bool {
	return !math.IsInf(v, 0) && !math.IsNaN(v)
}
