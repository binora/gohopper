package ev

import "gohopper/core/storage"

// IntsRefEdgeIntAccess wraps an IntsRef as an EdgeIntAccess.
// It ignores the edgeID parameter and accesses the underlying
// int array directly by index.
type IntsRefEdgeIntAccess struct {
	intsRef *storage.IntsRef
}

// NewIntsRefEdgeIntAccess creates a new IntsRefEdgeIntAccess backed by the
// given IntsRef.
func NewIntsRefEdgeIntAccess(intsRef *storage.IntsRef) *IntsRefEdgeIntAccess {
	return &IntsRefEdgeIntAccess{intsRef: intsRef}
}

func (a *IntsRefEdgeIntAccess) GetInt(_ int, index int) int32 {
	return a.intsRef.Ints[index]
}

func (a *IntsRefEdgeIntAccess) SetInt(_ int, index int, value int32) {
	a.intsRef.Ints[index] = value
}
