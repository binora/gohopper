package ch

import (
	"fmt"
	"math"
)

// NodeBasedWitnessPathSearcher performs witness searches during node-based CH preparation.
// Witness searches at node B determine if there is a path between two neighbor nodes A and C
// when we exclude B and check if this path is shorter than or equal to A-B-C.
type NodeBasedWitnessPathSearcher struct {
	outEdgeExplorer PrepareGraphEdgeExplorer
	weights         []float64
	changedNodes    []int
	heap            *IntFloatBinaryHeap
	ignoreNode      int
	settledNodes    int
}

func NewNodeBasedWitnessPathSearcher(graph *CHPreparationGraph) *NodeBasedWitnessPathSearcher {
	weights := make([]float64, graph.GetNodes())
	for i := range weights {
		weights[i] = math.Inf(1)
	}
	return &NodeBasedWitnessPathSearcher{
		outEdgeExplorer: graph.CreateOutEdgeExplorer(),
		weights:         weights,
		changedNodes:    make([]int, 0, 1000),
		heap:            NewIntFloatBinaryHeap(1000),
		ignoreNode:      -1,
	}
}

// Init sets up a search for the given start node and an ignored node. The shortest path tree
// will be re-used for different target nodes until this method is called again.
func (s *NodeBasedWitnessPathSearcher) Init(startNode, ignoreNode int) {
	s.reset()
	s.ignoreNode = ignoreNode
	s.weights[startNode] = 0
	s.changedNodes = append(s.changedNodes, startNode)
	s.heap.Insert(0, startNode)
}

// FindUpperBound runs or continues a Dijkstra search starting at the startNode and ignoring the
// ignoreNode given in Init(). Returns an upper bound for the real shortest path weight.
func (s *NodeBasedWitnessPathSearcher) FindUpperBound(targetNode int, acceptedWeight float64, maxSettledNodes int) float64 {
	for !s.heap.IsEmpty() && s.settledNodes < maxSettledNodes && float64(s.heap.PeekKey()) <= acceptedWeight {
		if s.weights[targetNode] <= acceptedWeight {
			return s.weights[targetNode]
		}
		node := s.heap.Poll()
		iter := s.outEdgeExplorer.SetBaseNode(node)
		for iter.Next() {
			adjNode := iter.GetAdjNode()
			if adjNode == s.ignoreNode {
				continue
			}
			weight := s.weights[node] + iter.GetWeight()
			if math.IsInf(weight, 1) {
				continue
			}
			adjWeight := s.weights[adjNode]
			if math.IsInf(adjWeight, 1) {
				s.weights[adjNode] = weight
				s.heap.Insert(weight, adjNode)
				s.changedNodes = append(s.changedNodes, adjNode)
			} else if weight < adjWeight {
				s.weights[adjNode] = weight
				s.heap.Update(weight, adjNode)
			}
		}
		s.settledNodes++
		if node == targetNode {
			return s.weights[node]
		}
	}
	return s.weights[targetNode]
}

func (s *NodeBasedWitnessPathSearcher) GetSettledNodes() int {
	return s.settledNodes
}

func (s *NodeBasedWitnessPathSearcher) reset() {
	for _, node := range s.changedNodes {
		s.weights[node] = math.Inf(1)
	}
	s.changedNodes = s.changedNodes[:0]
	s.heap.Clear()
	s.ignoreNode = -1
	s.settledNodes = 0
}

// GetMemoryUsageAsString returns currently used memory in MB (approximately).
func (s *NodeBasedWitnessPathSearcher) GetMemoryUsageAsString() string {
	mb := (8*int64(len(s.weights)) + int64(len(s.changedNodes))*4 + s.heap.GetMemoryUsage()) / (1 << 20)
	return fmt.Sprintf("%dMB", mb)
}
