package util

// EdgeExplorer provides an EdgeIterator for the edges of a given node.
// Create via graph.CreateEdgeExplorer(). Use one instance per goroutine.
type EdgeExplorer interface {
	SetBaseNode(baseNode int) EdgeIterator
}
