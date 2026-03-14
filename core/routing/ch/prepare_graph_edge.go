package ch

type PrepareGraphEdgeExplorer interface {
	SetBaseNode(node int) PrepareGraphEdgeIterator
}

type PrepareGraphEdgeIterator interface {
	Next() bool
	GetBaseNode() int
	GetAdjNode() int
	GetPrepareEdge() int
	IsShortcut() bool
	GetOrigEdgeKeyFirst() int
	GetOrigEdgeKeyLast() int
	GetSkipped1() int
	GetSkipped2() int
	GetWeight() float64
	GetOrigEdgeCount() int
	SetSkippedEdges(skipped1, skipped2 int)
	SetWeight(weight float64)
	SetOrigEdgeCount(origEdgeCount int)
}

type PrepareGraphOrigEdgeExplorer interface {
	SetBaseNode(node int) PrepareGraphOrigEdgeIterator
}

type PrepareGraphOrigEdgeIterator interface {
	Next() bool
	GetBaseNode() int
	GetAdjNode() int
	GetOrigEdgeKeyFirst() int
	GetOrigEdgeKeyLast() int
}
