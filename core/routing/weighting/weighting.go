package weighting

import (
	"gohopper/core/util"
	"regexp"
)

var validNamePattern = regexp.MustCompile(`^[|_a-z]+$`)

// Weighting specifies how the best route is calculated.
type Weighting interface {
	// CalcMinWeightPerDistance returns the minimal weight per meter.
	// Used only for the heuristic estimation in A*.
	// E.g. for fastest-route this returns 1/max_velocity.
	CalcMinWeightPerDistance() float64

	// CalcEdgeWeight calculates the weight of a given edge.
	// A high value indicates the edge should be avoided.
	// Must return a value in [0, +Inf). Must not return NaN.
	CalcEdgeWeight(edgeState util.EdgeIteratorState, reverse bool) float64

	// CalcEdgeMillis calculates the time in milliseconds to travel along the edge.
	CalcEdgeMillis(edgeState util.EdgeIteratorState, reverse bool) int64

	// CalcTurnWeight returns the turn weight for transitioning from inEdge to outEdge at viaNode.
	CalcTurnWeight(inEdge, viaNode, outEdge int) float64

	// CalcTurnMillis returns the turn time in milliseconds.
	CalcTurnMillis(inEdge, viaNode, outEdge int) int64

	// HasTurnCosts reports whether this weighting returns non-zero turn costs.
	HasTurnCosts() bool

	// GetName returns the name of this weighting.
	GetName() string
}

// IsValidName checks whether the given name is a valid weighting name.
// Valid names consist only of lowercase letters, underscores, and pipes.
func IsValidName(name string) bool {
	if name == "" {
		return false
	}
	return validNamePattern.MatchString(name)
}
