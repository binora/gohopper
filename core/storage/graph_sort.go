package storage

import (
	"cmp"
	"log"
	"slices"

	routingutil "gohopper/core/routing/util"
	"gohopper/core/util"
)

// hilbertOrder controls the Hilbert curve grid resolution (2^order x 2^order).
const hilbertOrder = 31

// SortGraphAlongHilbertCurve reorders graph nodes and edges along a Hilbert curve
// for improved CPU cache locality during routing.
func SortGraphAlongHilbertCurve(graph *BaseGraph) {
	log.Println("sorting graph along Hilbert curve...")

	na := graph.GetNodeAccess()
	nodeCount := graph.GetNodes()
	edgeCount := graph.GetEdges()

	hilbertIndices := make([]int64, nodeCount)
	for node := range nodeCount {
		hilbertIndices[node] = util.LatLonToHilbertIndex(na.GetLat(node), na.GetLon(node), hilbertOrder)
	}

	nodeOrder := identityPermutation(nodeCount)
	slices.SortStableFunc(nodeOrder, func(a, b int) int {
		return cmp.Compare(hilbertIndices[a], hilbertIndices[b])
	})

	edgeOrder := buildEdgeOrder(graph, nodeOrder, edgeCount)

	SortGraphForGivenOrdering(graph, invertPermutation(nodeOrder), invertPermutation(edgeOrder))
}

// buildEdgeOrder determines edge ordering by traversing nodes in Hilbert order
// and collecting edges as they are first encountered.
func buildEdgeOrder(graph *BaseGraph, nodeOrder []int, edgeCount int) []int {
	explorer := graph.CreateEdgeExplorer(routingutil.AllEdges)
	edgeOrder := make([]int, 0, edgeCount)
	seen := make([]bool, edgeCount)
	for _, node := range nodeOrder {
		iter := explorer.SetBaseNode(node)
		for iter.Next() {
			edge := iter.GetEdge()
			if !seen[edge] {
				edgeOrder = append(edgeOrder, edge)
				seen[edge] = true
			}
		}
	}
	return edgeOrder
}

// SortGraphForGivenOrdering reorders edges and relabels nodes using the given mappings.
// Both slices map old IDs to new IDs.
func SortGraphForGivenOrdering(graph *BaseGraph, newNodesByOld, newEdgesByOld []int) {
	log.Println("sorting graph for fixed ordering...")

	graph.SortEdges(func(old int) int { return newEdgesByOld[old] })
	graph.RelabelNodes(func(old int) int { return newNodesByOld[old] })
}

// identityPermutation returns [0, 1, 2, ..., n-1].
func identityPermutation(n int) []int {
	perm := make([]int, n)
	for i := range perm {
		perm[i] = i
	}
	return perm
}

// invertPermutation creates an inverse mapping: if perm[i] = j, then result[j] = i.
func invertPermutation(perm []int) []int {
	inv := make([]int, len(perm))
	for i, v := range perm {
		inv[v] = i
	}
	return inv
}
