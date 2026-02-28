package storage

import "gohopper/core/routing/ev"

var _ ev.EdgeIntAccess = (*IntsRefEdgeIntAccess)(nil)

// IntsRefEdgeIntAccess wraps an IntsRef as an ev.EdgeIntAccess,
// ignoring the edgeID parameter.
type IntsRefEdgeIntAccess struct {
	ref *IntsRef
}

func NewIntsRefEdgeIntAccess(ref *IntsRef) *IntsRefEdgeIntAccess {
	return &IntsRefEdgeIntAccess{ref: ref}
}

func (a *IntsRefEdgeIntAccess) GetInt(_ int, index int) int32 {
	return a.ref.Ints[index]
}

func (a *IntsRefEdgeIntAccess) SetInt(_ int, index int, value int32) {
	a.ref.Ints[index] = value
}
