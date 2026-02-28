package util

const (
	NoEdge  = -1
	AnyEdge = -2
)

// EdgeIterator iterates through edges of one node.
type EdgeIterator interface {
	EdgeIteratorState
	Next() bool
}

func EdgeIsValid(edgeID int) bool {
	return edgeID >= 0
}
