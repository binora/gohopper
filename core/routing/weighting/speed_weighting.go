package weighting

import (
	"math"

	"gohopper/core/routing/ev"
	"gohopper/core/util"
)

// Compile-time check that SpeedWeighting implements Weighting.
var _ Weighting = (*SpeedWeighting)(nil)

// SpeedWeighting calculates edge weights based on distance / speed.
type SpeedWeighting struct {
	speedEnc         ev.DecimalEncodedValue
	turnCostProvider TurnCostProvider
}

// NewSpeedWeighting creates a SpeedWeighting with no turn costs.
func NewSpeedWeighting(speedEnc ev.DecimalEncodedValue) *SpeedWeighting {
	return NewSpeedWeightingWithProvider(speedEnc, NoTurnCostProvider)
}

// NewSpeedWeightingWithProvider creates a SpeedWeighting with the given TurnCostProvider.
func NewSpeedWeightingWithProvider(speedEnc ev.DecimalEncodedValue, tcp TurnCostProvider) *SpeedWeighting {
	return &SpeedWeighting{
		speedEnc:         speedEnc,
		turnCostProvider: tcp,
	}
}

func (w *SpeedWeighting) CalcMinWeightPerDistance() float64 {
	return 1.0 / w.speedEnc.GetMaxStorableDecimal()
}

func (w *SpeedWeighting) CalcEdgeWeight(edgeState util.EdgeIteratorState, reverse bool) float64 {
	var speed float64
	if reverse {
		speed = edgeState.GetReverseDecimal(w.speedEnc)
	} else {
		speed = edgeState.GetDecimal(w.speedEnc)
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
