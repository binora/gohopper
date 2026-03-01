package weighting

import (
	"fmt"

	"gohopper/core/util"
)

// AbstractAdjustedWeighting wraps another Weighting and delegates all methods
// to it. Embed this struct and override specific methods to adjust behavior.
type AbstractAdjustedWeighting struct {
	SuperWeighting Weighting
}

func NewAbstractAdjustedWeighting(superWeighting Weighting) AbstractAdjustedWeighting {
	if superWeighting == nil {
		panic("no super weighting set")
	}
	return AbstractAdjustedWeighting{SuperWeighting: superWeighting}
}

func (w *AbstractAdjustedWeighting) CalcMinWeightPerDistance() float64 {
	return w.SuperWeighting.CalcMinWeightPerDistance()
}

func (w *AbstractAdjustedWeighting) CalcEdgeWeight(edgeState util.EdgeIteratorState, reverse bool) float64 {
	return w.SuperWeighting.CalcEdgeWeight(edgeState, reverse)
}

func (w *AbstractAdjustedWeighting) CalcEdgeMillis(edgeState util.EdgeIteratorState, reverse bool) int64 {
	return w.SuperWeighting.CalcEdgeMillis(edgeState, reverse)
}

func (w *AbstractAdjustedWeighting) CalcTurnWeight(inEdge, viaNode, outEdge int) float64 {
	return w.SuperWeighting.CalcTurnWeight(inEdge, viaNode, outEdge)
}

func (w *AbstractAdjustedWeighting) CalcTurnMillis(inEdge, viaNode, outEdge int) int64 {
	return w.SuperWeighting.CalcTurnMillis(inEdge, viaNode, outEdge)
}

func (w *AbstractAdjustedWeighting) HasTurnCosts() bool {
	return w.SuperWeighting.HasTurnCosts()
}

func (w *AbstractAdjustedWeighting) GetName() string {
	return w.SuperWeighting.GetName()
}

func (w *AbstractAdjustedWeighting) String() string {
	return fmt.Sprintf("%s|%s", w.GetName(), w.SuperWeighting)
}
