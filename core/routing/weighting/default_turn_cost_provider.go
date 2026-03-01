package weighting

import (
	"math"

	"gohopper/core/routing/ev"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// Compile-time check that DefaultTurnCostProvider implements TurnCostProvider.
var _ TurnCostProvider = (*DefaultTurnCostProvider)(nil)

// DefaultTurnCostProvider reads turn restrictions from TurnCostStorage and
// applies configurable u-turn costs.
type DefaultTurnCostProvider struct {
	turnRestrictionEnc ev.BooleanEncodedValue
	turnCostStorage    *storage.TurnCostStorage
	nodeAccess         storage.NodeAccess
	uTurnCosts         float64
}

// NewDefaultTurnCostProvider creates a DefaultTurnCostProvider.
// A negative uTurnCosts value means u-turns are infinitely expensive.
// turnRestrictionEnc may be nil, in which case no turn restrictions are checked.
func NewDefaultTurnCostProvider(
	turnRestrictionEnc ev.BooleanEncodedValue,
	turnCostStorage *storage.TurnCostStorage,
	nodeAccess storage.NodeAccess,
	uTurnCosts int,
) *DefaultTurnCostProvider {
	cost := float64(uTurnCosts)
	if uTurnCosts < 0 {
		cost = math.Inf(1)
	}
	return &DefaultTurnCostProvider{
		turnRestrictionEnc: turnRestrictionEnc,
		turnCostStorage:    turnCostStorage,
		nodeAccess:         nodeAccess,
		uTurnCosts:         cost,
	}
}

// CalcTurnWeight returns the turn weight for transitioning from inEdge to outEdge
// at viaNode. Returns +Inf for restricted turns, the configured u-turn cost for
// u-turns, and 0 otherwise.
func (p *DefaultTurnCostProvider) CalcTurnWeight(inEdge, viaNode, outEdge int) float64 {
	if !util.EdgeIsValid(inEdge) || !util.EdgeIsValid(outEdge) {
		return 0
	}
	if inEdge == outEdge {
		return p.uTurnCosts
	}
	if p.turnRestrictionEnc != nil &&
		p.turnCostStorage.GetBool(p.nodeAccess, p.turnRestrictionEnc, inEdge, viaNode, outEdge) {
		return math.Inf(1)
	}
	return 0
}

// CalcTurnMillis always returns 0. Making a proper assumption about turn time
// is very hard; zero is the simplest approach.
func (p *DefaultTurnCostProvider) CalcTurnMillis(inEdge, viaNode, outEdge int) int64 {
	return 0
}
