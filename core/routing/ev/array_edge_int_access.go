package ev

var _ EdgeIntAccess = (*ArrayEdgeIntAccess)(nil)

// ArrayEdgeIntAccess stores edge int32 values in a dynamically-growing slice,
// laid out as intsPerEdge consecutive slots per edge.
type ArrayEdgeIntAccess struct {
	intsPerEdge int
	arr         []int32
}

func NewArrayEdgeIntAccess(intsPerEdge int) *ArrayEdgeIntAccess {
	return &ArrayEdgeIntAccess{intsPerEdge: intsPerEdge}
}

func NewArrayEdgeIntAccessFromBytes(bytes int) *ArrayEdgeIntAccess {
	return NewArrayEdgeIntAccess((bytes + 3) / 4)
}

func (a *ArrayEdgeIntAccess) GetInt(edgeID, index int) int32 {
	i := edgeID*a.intsPerEdge + index
	if i >= len(a.arr) {
		return 0
	}
	return a.arr[i]
}

func (a *ArrayEdgeIntAccess) SetInt(edgeID, index int, value int32) {
	i := edgeID*a.intsPerEdge + index
	if i >= len(a.arr) {
		grown := make([]int32, i+1)
		copy(grown, a.arr)
		a.arr = grown
	}
	a.arr[i] = value
}
