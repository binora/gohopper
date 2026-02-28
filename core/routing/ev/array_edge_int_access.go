package ev

// ArrayEdgeIntAccess stores edge integer values in a dynamically-growing
// slice, laid out as intsPerEdge consecutive int32 slots per edge.
type ArrayEdgeIntAccess struct {
	intsPerEdge int
	arr         []int32
}

// NewArrayEdgeIntAccess creates a new ArrayEdgeIntAccess with the given
// number of int32 slots per edge.
func NewArrayEdgeIntAccess(intsPerEdge int) *ArrayEdgeIntAccess {
	return &ArrayEdgeIntAccess{intsPerEdge: intsPerEdge}
}

// NewArrayEdgeIntAccessFromBytes creates a new ArrayEdgeIntAccess with enough
// int32 slots to hold the given number of bytes per edge.
func NewArrayEdgeIntAccessFromBytes(bytes int) *ArrayEdgeIntAccess {
	return NewArrayEdgeIntAccess((bytes + 3) / 4)
}

func (a *ArrayEdgeIntAccess) GetInt(edgeID, index int) int32 {
	arrIndex := edgeID*a.intsPerEdge + index
	if arrIndex >= len(a.arr) {
		return 0
	}
	return a.arr[arrIndex]
}

func (a *ArrayEdgeIntAccess) SetInt(edgeID, index int, value int32) {
	arrIndex := edgeID*a.intsPerEdge + index
	if arrIndex >= len(a.arr) {
		grown := make([]int32, arrIndex+1)
		copy(grown, a.arr)
		a.arr = grown
	}
	a.arr[arrIndex] = value
}
