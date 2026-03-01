package routing

import (
	"math"

	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/util"
)

// NewFiniteWeightFilter returns an EdgeFilter that accepts edges whose weight
// is finite (not +-Inf and not NaN) in at least one direction.
func NewFiniteWeightFilter(w weighting.Weighting) routingutil.EdgeFilter {
	return func(edgeState util.EdgeIteratorState) bool {
		return isFinite(w.CalcEdgeWeight(edgeState, false)) ||
			isFinite(w.CalcEdgeWeight(edgeState, true))
	}
}

// isFinite mirrors Java's Double.isFinite: true iff v is neither infinite nor NaN.
func isFinite(v float64) bool {
	return !math.IsInf(v, 0) && !math.IsNaN(v)
}
