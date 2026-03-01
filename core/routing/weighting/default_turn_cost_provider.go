package weighting

import (
	"math"

	"gohopper/core/routing/ev"
	"gohopper/core/storage"
	"gohopper/core/util"
)

var _ TurnCostProvider = (*DefaultTurnCostProvider)(nil)

// DefaultTurnCostProvider reads turn restrictions from TurnCostStorage and
// applies configurable u-turn costs.
type DefaultTurnCostProvider struct {
	turnRestrictionEnc ev.BooleanEncodedValue
	turnCostStorage    *storage.TurnCostStorage
	nodeAccess         storage.NodeAccess
	uTurnCosts         float64
}

// NewDefaultTurnCostProvider creates a provider that reads turn restrictions
// from storage. A negative uTurnCosts means u-turns are infinitely expensive.
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

func (p *DefaultTurnCostProvider) CalcTurnMillis(_, _, _ int) int64 {
	return 0
}
