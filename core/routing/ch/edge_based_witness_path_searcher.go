package ch

import (
	"fmt"
	"math"

	"gohopper/core/util"
)

const (
	noNode            = -1
	maxZeroWeightLoop = 1e-3
)

// EdgeBasedWitnessPathSearcherStats tracks performance statistics for edge-based witness path searches.
type EdgeBasedWitnessPathSearcherStats struct {
	NumTrees    int64
	NumSearches int64
	NumPolls    int64
	MaxPolls    int64
	NumExplored int64
	MaxExplored int64
	NumUpdates  int64
	MaxUpdates  int64
	NumCapped   int64
}

func (s *EdgeBasedWitnessPathSearcherStats) String() string {
	return fmt.Sprintf(
		"trees: %12d, searches: %15d, capped: %12d (%5.2f%%), polled: avg %s max %6d, explored: avg %s max %6d, updated: avg %s max %6d",
		s.NumTrees, s.NumSearches, s.NumCapped,
		100*float64(s.NumCapped)/float64(s.NumSearches),
		quotient(s.NumPolls, s.NumTrees), s.MaxPolls,
		quotient(s.NumExplored, s.NumTrees), s.MaxExplored,
		quotient(s.NumUpdates, s.NumTrees), s.MaxUpdates,
	)
}

func quotient(a, b int64) string {
	if b == 0 {
		return "  NaN"
	}
	return fmt.Sprintf("%5.1f", float64(a)/float64(b))
}

// EdgeBasedWitnessPathSearcher performs local witness path searches for edge-based CH preparation.
type EdgeBasedWitnessPathSearcher struct {
	prepareGraph       *CHPreparationGraph
	outEdgeExplorer    PrepareGraphEdgeExplorer
	origInEdgeExplorer PrepareGraphOrigEdgeExplorer

	sourceNode int
	centerNode int

	numPolls   int
	numUpdates int

	weights                  []float64
	parents                  []int
	adjNodesAndIsPathToCenters []int
	changedEdgeKeys          []int
	dijkstraHeap             *IntFloatBinaryHeap

	stats *EdgeBasedWitnessPathSearcherStats
}

func NewEdgeBasedWitnessPathSearcher(prepareGraph *CHPreparationGraph) *EdgeBasedWitnessPathSearcher {
	s := &EdgeBasedWitnessPathSearcher{
		prepareGraph:       prepareGraph,
		outEdgeExplorer:    prepareGraph.CreateOutEdgeExplorer(),
		origInEdgeExplorer: prepareGraph.CreateInOrigEdgeExplorer(),
	}
	s.initStorage(2 * prepareGraph.GetOriginalEdges())
	s.initCollections()
	return s
}

// InitSearch deletes the shortest path tree found so far and initializes a new witness path search.
func (s *EdgeBasedWitnessPathSearcher) InitSearch(sourceEdgeKey, sourceNode, centerNode int, stats *EdgeBasedWitnessPathSearcherStats) {
	s.stats = stats
	stats.NumTrees++
	s.sourceNode = sourceNode
	s.centerNode = centerNode

	s.weights[sourceEdgeKey] = 0
	s.parents[sourceEdgeKey] = -1
	s.setAdjNodeAndPathToCenter(sourceEdgeKey, sourceNode, true)
	s.changedEdgeKeys = append(s.changedEdgeKeys, sourceEdgeKey)
	s.dijkstraHeap.Insert(0, sourceEdgeKey)
}

// RunSearch runs a witness path search for a given target edge key. Results of previous searches
// are reused and the previous search is extended if necessary.
func (s *EdgeBasedWitnessPathSearcher) RunSearch(targetNode, targetEdgeKey int, acceptedWeight float64, maxPolls int) float64 {
	s.stats.NumSearches++

	// first check if we can already reach the target edge from the existing shortest path tree
	inIter := s.origInEdgeExplorer.SetBaseNode(targetNode)
	for inIter.Next() {
		edgeKey := util.ReverseEdgeKey(inIter.GetOrigEdgeKeyLast())
		if math.IsInf(s.weights[edgeKey], 1) {
			continue
		}
		weight := s.weights[edgeKey] + s.calcTurnWeight(edgeKey, targetNode, targetEdgeKey)
		if weight < acceptedWeight || (weight == acceptedWeight && (s.parents[edgeKey] < 0 || !s.isPathToCenter(s.parents[edgeKey]))) {
			return weight
		}
	}

	// run the search
	for !s.dijkstraHeap.IsEmpty() && s.numPolls < maxPolls &&
		s.weights[s.dijkstraHeap.PeekElement()] < acceptedWeight {

		currKey := s.dijkstraHeap.Poll()
		s.numPolls++
		currNode := s.getAdjNode(currKey)
		iter := s.outEdgeExplorer.SetBaseNode(currNode)
		foundWeight := math.Inf(1)
		for iter.Next() {
			if currNode == s.sourceNode && iter.GetAdjNode() == s.sourceNode && iter.GetWeight() < maxZeroWeightLoop {
				continue
			}
			weight := s.weights[currKey] + s.calcTurnWeight(currKey, currNode, iter.GetOrigEdgeKeyFirst()) + iter.GetWeight()
			if math.IsInf(weight, 1) {
				continue
			}
			key := iter.GetOrigEdgeKeyLast()
			adjNode := iter.GetAdjNode()
			isPathToCenter := s.isPathToCenter(currKey) && adjNode == s.centerNode
			updated := false
			if math.IsInf(s.weights[key], 1) {
				s.weights[key] = weight
				s.parents[key] = currKey
				s.setAdjNodeAndPathToCenter(key, adjNode, isPathToCenter)
				s.changedEdgeKeys = append(s.changedEdgeKeys, key)
				s.dijkstraHeap.Insert(weight, key)
				updated = true
			} else if weight < s.weights[key] ||
				(weight == s.weights[key] && !s.isPathToCenter(currKey)) {
				s.numUpdates++
				s.weights[key] = weight
				s.parents[key] = currKey
				s.setAdjNodeAndPathToCenter(key, adjNode, isPathToCenter)
				s.dijkstraHeap.Update(weight, key)
				updated = true
			}
			if updated && adjNode == targetNode && (!s.isPathToCenter(currKey) || s.parents[currKey] < 0) {
				foundWeight = min(foundWeight, weight+s.calcTurnWeight(key, targetNode, targetEdgeKey))
			}
		}
		if foundWeight <= acceptedWeight {
			return foundWeight
		}
	}
	if s.numPolls == maxPolls {
		s.stats.NumCapped++
	}
	return math.Inf(1)
}

// FinishSearch records stats and resets the search state.
func (s *EdgeBasedWitnessPathSearcher) FinishSearch() {
	polls := int64(s.numPolls)
	explored := int64(len(s.changedEdgeKeys))
	updates := int64(s.numUpdates)

	s.stats.NumPolls += polls
	s.stats.MaxPolls = max(s.stats.MaxPolls, polls)
	s.stats.NumExplored += explored
	s.stats.MaxExplored = max(s.stats.MaxExplored, explored)
	s.stats.NumUpdates += updates
	s.stats.MaxUpdates = max(s.stats.MaxUpdates, updates)

	s.reset()
}

func (s *EdgeBasedWitnessPathSearcher) Close() {
	s.prepareGraph.Close()
	s.outEdgeExplorer = nil
	s.origInEdgeExplorer = nil
	s.weights = nil
	s.parents = nil
	s.adjNodesAndIsPathToCenters = nil
	s.changedEdgeKeys = nil
	s.dijkstraHeap = nil
}

func (s *EdgeBasedWitnessPathSearcher) setAdjNodeAndPathToCenter(key, adjNode int, isPathToCenter bool) {
	val := adjNode << 1
	if isPathToCenter {
		val |= 1
	}
	s.adjNodesAndIsPathToCenters[key] = val
}

func (s *EdgeBasedWitnessPathSearcher) getAdjNode(key int) int {
	return s.adjNodesAndIsPathToCenters[key] >> 1
}

func (s *EdgeBasedWitnessPathSearcher) isPathToCenter(key int) bool {
	return s.adjNodesAndIsPathToCenters[key]&1 == 1
}

func (s *EdgeBasedWitnessPathSearcher) initStorage(numEntries int) {
	s.weights = make([]float64, numEntries)
	for i := range s.weights {
		s.weights[i] = math.Inf(1)
	}
	s.parents = make([]int, numEntries)
	for i := range s.parents {
		s.parents[i] = noNode
	}
	s.adjNodesAndIsPathToCenters = make([]int, numEntries)
	for i := range s.adjNodesAndIsPathToCenters {
		s.adjNodesAndIsPathToCenters[i] = noNode << 1
	}
}

func (s *EdgeBasedWitnessPathSearcher) initCollections() {
	s.changedEdgeKeys = make([]int, 0, 1000)
	s.dijkstraHeap = NewIntFloatBinaryHeap(1000)
}

func (s *EdgeBasedWitnessPathSearcher) reset() {
	s.numPolls = 0
	s.numUpdates = 0
	s.resetShortestPathTree()
}

func (s *EdgeBasedWitnessPathSearcher) resetShortestPathTree() {
	for _, key := range s.changedEdgeKeys {
		s.resetEntry(key)
	}
	s.changedEdgeKeys = s.changedEdgeKeys[:0]
	s.dijkstraHeap.Clear()
}

func (s *EdgeBasedWitnessPathSearcher) resetEntry(key int) {
	s.weights[key] = math.Inf(1)
	s.parents[key] = noNode
	s.setAdjNodeAndPathToCenter(key, noNode, false)
}

func (s *EdgeBasedWitnessPathSearcher) calcTurnWeight(inEdgeKey, viaNode, outEdgeKey int) float64 {
	return s.prepareGraph.GetTurnWeight(inEdgeKey, viaNode, outEdgeKey)
}
