package custom

import (
	"math"

	"gohopper/core/util"
)

const (
	Name      = "custom"
	SpeedConv = 3.6
)

// TurnCostProvider calculates turn costs.
// Duplicates weighting.TurnCostProvider to avoid a circular import.
type TurnCostProvider interface {
	CalcTurnWeight(inEdge, viaNode, outEdge int) float64
	CalcTurnMillis(inEdge, viaNode, outEdge int) int64
}

// Parameters holds the pre-computed inputs for constructing a CustomWeighting.
type Parameters struct {
	EdgeToSpeed       EdgeToDoubleMapping
	EdgeToPriority    EdgeToDoubleMapping
	MaxSpeed          float64
	MaxPriority       float64
	DistanceInfluence float64 // seconds per kilometer (converted to s/m internally)
	HeadingPenalty    float64 // seconds
}

type CustomWeighting struct {
	edgeToSpeed       EdgeToDoubleMapping
	edgeToPriority    EdgeToDoubleMapping
	maxSpeed          float64
	maxPriority       float64
	distanceInfluence float64 // stored in s/m
	headingPenalty    float64
	turnCostProvider  TurnCostProvider
	hasTurnCosts      bool
}

func NewCustomWeighting(turnCostProvider TurnCostProvider, hasTurnCosts bool, params *Parameters) *CustomWeighting {
	di := params.DistanceInfluence / 1000.0
	if di < 0 {
		panic("distance_influence cannot be negative")
	}
	return &CustomWeighting{
		edgeToSpeed:       params.EdgeToSpeed,
		edgeToPriority:    params.EdgeToPriority,
		maxSpeed:          params.MaxSpeed,
		maxPriority:       params.MaxPriority,
		distanceInfluence: di,
		headingPenalty:    params.HeadingPenalty,
		turnCostProvider:  turnCostProvider,
		hasTurnCosts:      hasTurnCosts,
	}
}

func (w *CustomWeighting) CalcMinWeightPerDistance() float64 {
	return 1.0/(w.maxSpeed/SpeedConv)/w.maxPriority + w.distanceInfluence
}

func (w *CustomWeighting) CalcEdgeWeight(edgeState util.EdgeIteratorState, reverse bool) float64 {
	priority := w.edgeToPriority(edgeState, reverse)
	if priority == 0 {
		return math.Inf(1)
	}

	distance := edgeState.GetDistance()
	seconds := w.calcSeconds(distance, edgeState, reverse)
	if math.IsInf(seconds, 1) {
		return math.Inf(1)
	}

	distanceCosts := distance * w.distanceInfluence
	if math.IsInf(distanceCosts, 1) {
		return math.Inf(1)
	}
	return seconds/priority + distanceCosts
}

func (w *CustomWeighting) calcSeconds(distance float64, edgeState util.EdgeIteratorState, reverse bool) float64 {
	speed := w.edgeToSpeed(edgeState, reverse)
	if speed < 0 {
		panic("speed cannot be negative")
	}
	if speed == 0 {
		return math.Inf(1)
	}
	return distance / speed * SpeedConv
}

func (w *CustomWeighting) CalcEdgeMillis(edgeState util.EdgeIteratorState, reverse bool) int64 {
	return int64(math.Round(w.calcSeconds(edgeState.GetDistance(), edgeState, reverse) * 1000))
}

func (w *CustomWeighting) CalcTurnWeight(inEdge, viaNode, outEdge int) float64 {
	return w.turnCostProvider.CalcTurnWeight(inEdge, viaNode, outEdge)
}

func (w *CustomWeighting) CalcTurnMillis(inEdge, viaNode, outEdge int) int64 {
	return w.turnCostProvider.CalcTurnMillis(inEdge, viaNode, outEdge)
}

func (w *CustomWeighting) HasTurnCosts() bool {
	return w.hasTurnCosts
}

func (w *CustomWeighting) GetName() string {
	return Name
}
