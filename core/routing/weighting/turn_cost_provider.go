package weighting

// TurnCostProvider defines how turn costs and turn times are calculated.
type TurnCostProvider interface {
	// CalcTurnWeight returns the turn weight for transitioning from inEdge to outEdge at viaNode.
	CalcTurnWeight(inEdge, viaNode, outEdge int) float64

	// CalcTurnMillis returns the time in milliseconds to take the turn.
	CalcTurnMillis(inEdge, viaNode, outEdge int) int64
}

// NoTurnCostProvider is a TurnCostProvider that always returns zero costs.
var NoTurnCostProvider TurnCostProvider = &noTurnCostProvider{}

type noTurnCostProvider struct{}

func (*noTurnCostProvider) CalcTurnWeight(inEdge, viaNode, outEdge int) float64 { return 0 }
func (*noTurnCostProvider) CalcTurnMillis(inEdge, viaNode, outEdge int) int64   { return 0 }
