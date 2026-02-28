package util

// EdgeExplorer provides an EdgeIterator for the edges of a given node.
// Create via graph.CreateEdgeExplorer(). Use one instance per goroutine.
type EdgeExplorer interface {
	// SetBaseNode returns an EdgeIterator for iterating the edges of baseNode.
	SetBaseNode(baseNode int) EdgeIterator
}
