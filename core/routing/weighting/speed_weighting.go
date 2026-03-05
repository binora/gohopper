package weighting

import (
	"math"

	"gohopper/core/routing/ev"
	"gohopper/core/storage"
	"gohopper/core/util"
)

var _ Weighting = (*SpeedWeighting)(nil)

// SpeedWeighting calculates edge weights based on distance / speed.
// When an accessEnc is provided, edges where access is false are blocked (infinite weight).
type SpeedWeighting struct {
	speedEnc         ev.DecimalEncodedValue
	accessEnc        ev.BooleanEncodedValue
	turnCostProvider TurnCostProvider
}

func NewSpeedWeighting(speedEnc ev.DecimalEncodedValue) *SpeedWeighting {
	return &SpeedWeighting{speedEnc: speedEnc, turnCostProvider: NoTurnCostProvider}
}

func NewSpeedWeightingWithAccess(speedEnc ev.DecimalEncodedValue, accessEnc ev.BooleanEncodedValue) *SpeedWeighting {
	return &SpeedWeighting{speedEnc: speedEnc, accessEnc: accessEnc, turnCostProvider: NoTurnCostProvider}
}

func NewSpeedWeightingWithProvider(speedEnc ev.DecimalEncodedValue, tcp TurnCostProvider) *SpeedWeighting {
	return &SpeedWeighting{speedEnc: speedEnc, turnCostProvider: tcp}
}

func NewSpeedWeightingFull(speedEnc ev.DecimalEncodedValue, accessEnc ev.BooleanEncodedValue, tcp TurnCostProvider) *SpeedWeighting {
	return &SpeedWeighting{speedEnc: speedEnc, accessEnc: accessEnc, turnCostProvider: tcp}
}

func NewSpeedWeightingWithTurnCosts(
	speedEnc ev.DecimalEncodedValue,
	turnCostEnc ev.DecimalEncodedValue,
	tcs *storage.TurnCostStorage,
	na storage.NodeAccess,
	uTurnCosts float64,
) *SpeedWeighting {
	return NewSpeedWeightingWithProvider(speedEnc, &decimalTurnCostProvider{
		turnCostEnc: turnCostEnc,
		tcs:         tcs,
		na:          na,
		uTurnCosts:  uTurnCosts,
	})
}

type decimalTurnCostProvider struct {
	turnCostEnc ev.DecimalEncodedValue
	tcs         *storage.TurnCostStorage
	na          storage.NodeAccess
	uTurnCosts  float64
}

func (p *decimalTurnCostProvider) CalcTurnWeight(inEdge, viaNode, outEdge int) float64 {
	if !util.EdgeIsValid(inEdge) || !util.EdgeIsValid(outEdge) {
		return 0
	}
	cost := p.tcs.GetDecimal(p.na, p.turnCostEnc, inEdge, viaNode, outEdge)
	if inEdge == outEdge {
		return math.Max(cost, p.uTurnCosts)
	}
	return cost
}

func (p *decimalTurnCostProvider) CalcTurnMillis(inEdge, viaNode, outEdge int) int64 {
	return int64(1000 * p.CalcTurnWeight(inEdge, viaNode, outEdge))
}

func (w *SpeedWeighting) CalcMinWeightPerDistance() float64 {
	return 1.0 / w.speedEnc.GetMaxStorableDecimal()
}

func (w *SpeedWeighting) CalcEdgeWeight(edgeState util.EdgeIteratorState, reverse bool) float64 {
	if w.accessEnc != nil {
		accessible := edgeState.GetBool(w.accessEnc)
		if reverse {
			accessible = edgeState.GetReverseBool(w.accessEnc)
		}
		if !accessible {
			return math.Inf(1)
		}
	}
	speed := edgeState.GetDecimal(w.speedEnc)
	if reverse {
		speed = edgeState.GetReverseDecimal(w.speedEnc)
	}
	if speed == 0 {
		return math.Inf(1)
	}
	return edgeState.GetDistance() / speed
}

func (w *SpeedWeighting) CalcEdgeMillis(edgeState util.EdgeIteratorState, reverse bool) int64 {
	return int64(1000 * w.CalcEdgeWeight(edgeState, reverse))
}

func (w *SpeedWeighting) CalcTurnWeight(inEdge, viaNode, outEdge int) float64 {
	return w.turnCostProvider.CalcTurnWeight(inEdge, viaNode, outEdge)
}

func (w *SpeedWeighting) CalcTurnMillis(inEdge, viaNode, outEdge int) int64 {
	return w.turnCostProvider.CalcTurnMillis(inEdge, viaNode, outEdge)
}

func (w *SpeedWeighting) HasTurnCosts() bool {
	return w.turnCostProvider != NoTurnCostProvider
}

func (w *SpeedWeighting) GetName() string {
	return "speed"
}
