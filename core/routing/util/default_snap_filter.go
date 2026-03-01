package util

import (
	"math"

	"gohopper/core/routing/ev"
	ghutil "gohopper/core/util"
)

// EdgeWeightCalc is the subset of Weighting needed by DefaultSnapFilter,
// avoiding an import cycle between routing/util and routing/weighting.
type EdgeWeightCalc interface {
	CalcEdgeWeight(edgeState ghutil.EdgeIteratorState, reverse bool) float64
}

// DefaultSnapFilter rejects edges belonging to a subnetwork or inaccessible
// according to the given weighting.
type DefaultSnapFilter struct {
	weighting       EdgeWeightCalc
	inSubnetworkEnc ev.BooleanEncodedValue
}

func NewDefaultSnapFilter(w EdgeWeightCalc, inSubnetworkEnc ev.BooleanEncodedValue) EdgeFilter {
	f := &DefaultSnapFilter{
		weighting:       w,
		inSubnetworkEnc: inSubnetworkEnc,
	}
	return f.Accept
}

func (f *DefaultSnapFilter) Accept(edgeState ghutil.EdgeIteratorState) bool {
	if edgeState.GetBool(f.inSubnetworkEnc) {
		return false
	}
	return isFinite(f.weighting.CalcEdgeWeight(edgeState, false)) ||
		isFinite(f.weighting.CalcEdgeWeight(edgeState, true))
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
