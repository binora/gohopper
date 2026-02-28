package storage

import "gohopper/core/routing/ev"

// IntsRefEdgeIntAccess wraps an IntsRef as an ev.EdgeIntAccess.
// It ignores the edgeID parameter and accesses the underlying
// int array directly by index.
type IntsRefEdgeIntAccess struct {
	intsRef *IntsRef
}

// NewIntsRefEdgeIntAccess creates a new IntsRefEdgeIntAccess backed by the
// given IntsRef.
func NewIntsRefEdgeIntAccess(intsRef *IntsRef) *IntsRefEdgeIntAccess {
	return &IntsRefEdgeIntAccess{intsRef: intsRef}
}

func (a *IntsRefEdgeIntAccess) GetInt(_ int, index int) int32 {
	return a.intsRef.Ints[index]
}

func (a *IntsRefEdgeIntAccess) SetInt(_ int, index int, value int32) {
	a.intsRef.Ints[index] = value
}

// Verify interface compliance.
var _ ev.EdgeIntAccess = (*IntsRefEdgeIntAccess)(nil)
