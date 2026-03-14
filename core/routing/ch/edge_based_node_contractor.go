package ch

import (
	"fmt"
	"math"
	"time"

	"gohopper/core/storage"
	"gohopper/core/util"
	webapi "gohopper/web-api"
)

type edgeBasedParams struct {
	edgeQuotientWeight         float32
	originalEdgeQuotientWeight float32
	hierarchyDepthWeight       float32
	maxPollFactorHeuristic     float64
	maxPollFactorContraction   float64
}

type edgeBasedStats struct {
	nodes    int
	duration time.Duration
}

func (s *edgeBasedStats) String() string {
	return fmt.Sprintf("time: %7.2fs, nodes: %10d", s.duration.Seconds(), s.nodes)
}

type edgeBasedShortcutHandler func(edgeFrom, edgeTo *PrepareCHEntry, origEdgeCount int)

// EdgeBasedNodeContractor implements NodeContractor for edge-based CH with turn cost support.
type EdgeBasedNodeContractor struct {
	prepareGraph                 *CHPreparationGraph
	inEdgeExplorer               PrepareGraphEdgeExplorer
	outEdgeExplorer              PrepareGraphEdgeExplorer
	existingShortcutExplorer     PrepareGraphEdgeExplorer
	sourceNodeOrigInEdgeExplorer PrepareGraphOrigEdgeExplorer
	chBuilder                    *storage.CHStorageBuilder
	params                       edgeBasedParams
	dijkstraDuration             time.Duration

	sourceNodes    map[int]struct{}
	addedShortcuts map[int64]struct{}

	addingStats   edgeBasedStats
	countingStats edgeBasedStats
	activeStats   *edgeBasedStats

	hierarchyDepths     []int
	witnessPathSearcher *EdgeBasedWitnessPathSearcher
	bridgePathFinder    *BridgePathFinder
	wpsStatsHeur        EdgeBasedWitnessPathSearcherStats
	wpsStatsContr       EdgeBasedWitnessPathSearcherStats

	addedShortcutsCount int

	numShortcuts     int
	numPrevEdges     int
	numOrigEdges     int
	numPrevOrigEdges int
	numAllEdges      int

	meanDegree float64
}

func NewEdgeBasedNodeContractor(prepareGraph *CHPreparationGraph, chBuilder *storage.CHStorageBuilder, pMap webapi.PMap) *EdgeBasedNodeContractor {
	c := &EdgeBasedNodeContractor{
		prepareGraph:   prepareGraph,
		chBuilder:      chBuilder,
		sourceNodes:    make(map[int]struct{}, 10),
		addedShortcuts: make(map[int64]struct{}),
		params: edgeBasedParams{
			edgeQuotientWeight:         100,
			originalEdgeQuotientWeight: 100,
			hierarchyDepthWeight:       20,
			maxPollFactorHeuristic:     4,
			maxPollFactorContraction:   200,
		},
	}
	c.extractParams(pMap)
	return c
}

func (c *EdgeBasedNodeContractor) extractParams(pMap webapi.PMap) {
	c.params.edgeQuotientWeight = float32(pMap.GetFloat64(EdgeQuotientWeight, float64(c.params.edgeQuotientWeight)))
	c.params.originalEdgeQuotientWeight = float32(pMap.GetFloat64(OriginalEdgeQuotientWeight, float64(c.params.originalEdgeQuotientWeight)))
	c.params.hierarchyDepthWeight = float32(pMap.GetFloat64(HierarchyDepthWeight, float64(c.params.hierarchyDepthWeight)))
	c.params.maxPollFactorHeuristic = pMap.GetFloat64(MaxPollFactorHeuristicEdge, c.params.maxPollFactorHeuristic)
	c.params.maxPollFactorContraction = pMap.GetFloat64(MaxPollFactorContractionEdge, c.params.maxPollFactorContraction)
}

func (c *EdgeBasedNodeContractor) InitFromGraph() {
	c.inEdgeExplorer = c.prepareGraph.CreateInEdgeExplorer()
	c.outEdgeExplorer = c.prepareGraph.CreateOutEdgeExplorer()
	c.existingShortcutExplorer = c.prepareGraph.CreateOutEdgeExplorer()
	c.sourceNodeOrigInEdgeExplorer = c.prepareGraph.CreateInOrigEdgeExplorer()
	c.hierarchyDepths = make([]int, c.prepareGraph.GetNodes())
	c.witnessPathSearcher = NewEdgeBasedWitnessPathSearcher(c.prepareGraph)
	c.bridgePathFinder = NewBridgePathFinder(c.prepareGraph)
	c.meanDegree = float64(c.prepareGraph.GetOriginalEdges()) / float64(c.prepareGraph.GetNodes())
}

func (c *EdgeBasedNodeContractor) CalculatePriority(node int) float32 {
	c.activeStats = &c.countingStats
	c.resetEdgeCounters()
	c.countPreviousEdges(node)
	if c.numAllEdges == 0 {
		return float32(math.Inf(-1))
	}
	start := time.Now()
	c.findAndHandlePrepareShortcuts(node, c.countShortcuts, int(c.meanDegree*c.params.maxPollFactorHeuristic), &c.wpsStatsHeur)
	c.activeStats.duration += time.Since(start)

	edgeQuotient := float32(c.numShortcuts) / float32(c.prepareGraph.GetDegree(node))
	origEdgeQuotient := float32(c.numOrigEdges) / float32(c.numPrevOrigEdges)
	hierarchyDepth := c.hierarchyDepths[node]
	return c.params.edgeQuotientWeight*edgeQuotient +
		c.params.originalEdgeQuotientWeight*origEdgeQuotient +
		c.params.hierarchyDepthWeight*float32(hierarchyDepth)
}

func (c *EdgeBasedNodeContractor) ContractNode(node int) []int {
	c.activeStats = &c.addingStats
	start := time.Now()
	c.findAndHandlePrepareShortcuts(node, func(edgeFrom, edgeTo *PrepareCHEntry, origEdgeCount int) {
		c.addShortcutsToPrepareGraph(edgeFrom, edgeTo, origEdgeCount)
	}, int(c.meanDegree*c.params.maxPollFactorContraction), &c.wpsStatsContr)
	c.insertShortcuts(node)
	neighbors := c.prepareGraph.Disconnect(node)
	c.meanDegree = (c.meanDegree*2 + float64(len(neighbors))) / 3
	c.updateHierarchyDepthsOfNeighbors(node, neighbors)
	c.activeStats.duration += time.Since(start)
	return neighbors
}

func (c *EdgeBasedNodeContractor) FinishContraction() {
	c.chBuilder.ReplaceSkippedEdges(c.prepareGraph.GetShortcutForPrepareEdge)
}

func (c *EdgeBasedNodeContractor) GetAddedShortcutsCount() int64 {
	return int64(c.addedShortcutsCount)
}

func (c *EdgeBasedNodeContractor) GetDijkstraSeconds() float32 {
	return float32(c.dijkstraDuration.Seconds())
}

func (c *EdgeBasedNodeContractor) GetStatisticsString() string {
	return fmt.Sprintf("degree_approx: %3.1f, priority   : %s, %s, contraction: %s, %s",
		c.meanDegree, &c.countingStats, &c.wpsStatsHeur, &c.addingStats, &c.wpsStatsContr)
}

func (c *EdgeBasedNodeContractor) findAndHandlePrepareShortcuts(node int, handler edgeBasedShortcutHandler, maxPolls int, wpsStats *EdgeBasedWitnessPathSearcherStats) {
	c.activeStats.nodes++
	clear(c.addedShortcuts)
	clear(c.sourceNodes)

	incomingEdges := c.inEdgeExplorer.SetBaseNode(node)
	for incomingEdges.Next() {
		sourceNode := incomingEdges.GetAdjNode()
		if sourceNode == node {
			continue
		}
		if _, exists := c.sourceNodes[sourceNode]; exists {
			continue
		}
		c.sourceNodes[sourceNode] = struct{}{}

		origInIter := c.sourceNodeOrigInEdgeExplorer.SetBaseNode(sourceNode)
		for origInIter.Next() {
			origInKey := util.ReverseEdgeKey(origInIter.GetOrigEdgeKeyLast())
			bridgePaths := c.bridgePathFinder.Find(origInKey, sourceNode, node)
			if len(bridgePaths) == 0 {
				continue
			}
			c.witnessPathSearcher.InitSearch(origInKey, sourceNode, node, wpsStats)
			for targetEdgeKey, bp := range bridgePaths {
				if math.IsInf(bp.Weight, 0) || math.IsNaN(bp.Weight) {
					panic("Bridge entry weights should always be finite")
				}
				dijkstraStart := time.Now()
				weight := c.witnessPathSearcher.RunSearch(bp.ChEntry.AdjNode, targetEdgeKey, bp.Weight, maxPolls)
				c.dijkstraDuration += time.Since(dijkstraStart)
				if weight <= bp.Weight {
					continue
				}
				root := bp.ChEntry
				for util.EdgeIsValid(root.Parent.PrepareEdge) {
					root = root.Parent
				}
				addedShortcutKey := util.BitLE.ToLongFromInts(int32(root.FirstEdgeKey), int32(bp.ChEntry.IncEdgeKey))
				if _, exists := c.addedShortcuts[addedShortcutKey]; exists {
					continue
				}
				c.addedShortcuts[addedShortcutKey] = struct{}{}
				initialTurnCost := c.prepareGraph.GetTurnWeight(origInKey, sourceNode, root.FirstEdgeKey)
				bp.ChEntry.Weight -= initialTurnCost
				handler(root, bp.ChEntry, bp.ChEntry.OrigEdges)
			}
			c.witnessPathSearcher.FinishSearch()
		}
	}
}

func (c *EdgeBasedNodeContractor) insertShortcuts(node int) {
	c.insertOutShortcuts(node)
	c.insertInShortcuts(node)
}

func (c *EdgeBasedNodeContractor) insertOutShortcuts(node int) {
	iter := c.outEdgeExplorer.SetBaseNode(node)
	for iter.Next() {
		if !iter.IsShortcut() {
			continue
		}
		shortcut := c.chBuilder.AddShortcutEdgeBased(node, iter.GetAdjNode(),
			ScFwdDir, iter.GetWeight(),
			iter.GetSkipped1(), iter.GetSkipped2(),
			iter.GetOrigEdgeKeyFirst(), iter.GetOrigEdgeKeyLast())
		c.prepareGraph.SetShortcutForPrepareEdge(iter.GetPrepareEdge(), c.prepareGraph.GetOriginalEdges()+shortcut)
		c.addedShortcutsCount++
	}
}

func (c *EdgeBasedNodeContractor) insertInShortcuts(node int) {
	iter := c.inEdgeExplorer.SetBaseNode(node)
	for iter.Next() {
		if !iter.IsShortcut() {
			continue
		}
		if iter.GetAdjNode() == node {
			continue
		}
		shortcut := c.chBuilder.AddShortcutEdgeBased(node, iter.GetAdjNode(),
			ScBwdDir, iter.GetWeight(),
			iter.GetSkipped1(), iter.GetSkipped2(),
			iter.GetOrigEdgeKeyFirst(), iter.GetOrigEdgeKeyLast())
		c.prepareGraph.SetShortcutForPrepareEdge(iter.GetPrepareEdge(), c.prepareGraph.GetOriginalEdges()+shortcut)
		c.addedShortcutsCount++
	}
}

func (c *EdgeBasedNodeContractor) countPreviousEdges(node int) {
	outIter := c.outEdgeExplorer.SetBaseNode(node)
	for outIter.Next() {
		c.numAllEdges++
		c.numPrevEdges++
		c.numPrevOrigEdges += outIter.GetOrigEdgeCount()
	}
	inIter := c.inEdgeExplorer.SetBaseNode(node)
	for inIter.Next() {
		c.numAllEdges++
		if inIter.GetBaseNode() == inIter.GetAdjNode() {
			continue
		}
		c.numPrevEdges++
		c.numPrevOrigEdges += inIter.GetOrigEdgeCount()
	}
}

func (c *EdgeBasedNodeContractor) updateHierarchyDepthsOfNeighbors(node int, neighbors []int) {
	level := c.hierarchyDepths[node]
	for _, n := range neighbors {
		if n == node {
			continue
		}
		if level+1 > c.hierarchyDepths[n] {
			c.hierarchyDepths[n] = level + 1
		}
	}
}

func (c *EdgeBasedNodeContractor) addShortcutsToPrepareGraph(edgeFrom, edgeTo *PrepareCHEntry, origEdgeCount int) *PrepareCHEntry {
	if edgeTo.Parent.PrepareEdge != edgeFrom.PrepareEdge {
		prev := c.addShortcutsToPrepareGraph(edgeFrom, edgeTo.Parent, origEdgeCount)
		return c.doAddShortcut(prev, edgeTo, origEdgeCount)
	}
	return c.doAddShortcut(edgeFrom, edgeTo, origEdgeCount)
}

func (c *EdgeBasedNodeContractor) doAddShortcut(edgeFrom, edgeTo *PrepareCHEntry, origEdgeCount int) *PrepareCHEntry {
	from := edgeFrom.Parent.AdjNode
	adjNode := edgeTo.AdjNode

	iter := c.existingShortcutExplorer.SetBaseNode(from)
	for iter.Next() {
		if !c.isSameShortcut(iter, adjNode, edgeFrom.FirstEdgeKey, edgeTo.IncEdgeKey) {
			continue
		}
		existingWeight := iter.GetWeight()
		if existingWeight <= edgeTo.Weight {
			entry := NewPrepareCHEntry(iter.GetPrepareEdge(), iter.GetOrigEdgeKeyFirst(), iter.GetOrigEdgeKeyLast(), adjNode, existingWeight, origEdgeCount)
			entry.Parent = edgeFrom.Parent
			return entry
		}
		iter.SetSkippedEdges(edgeFrom.PrepareEdge, edgeTo.PrepareEdge)
		iter.SetWeight(edgeTo.Weight)
		iter.SetOrigEdgeCount(origEdgeCount)
		entry := NewPrepareCHEntry(iter.GetPrepareEdge(), iter.GetOrigEdgeKeyFirst(), iter.GetOrigEdgeKeyLast(), adjNode, edgeTo.Weight, origEdgeCount)
		entry.Parent = edgeFrom.Parent
		return entry
	}

	origFirstKey := edgeFrom.FirstEdgeKey
	prepareEdge := c.prepareGraph.AddShortcut(from, adjNode, origFirstKey, edgeTo.IncEdgeKey, edgeFrom.PrepareEdge, edgeTo.PrepareEdge, edgeTo.Weight, origEdgeCount)
	entry := NewPrepareCHEntry(prepareEdge, origFirstKey, -1, edgeTo.AdjNode, edgeTo.Weight, origEdgeCount)
	entry.Parent = edgeFrom.Parent
	return entry
}

func (c *EdgeBasedNodeContractor) isSameShortcut(iter PrepareGraphEdgeIterator, adjNode, firstOrigEdgeKey, lastOrigEdgeKey int) bool {
	return iter.IsShortcut() &&
		iter.GetAdjNode() == adjNode &&
		iter.GetOrigEdgeKeyFirst() == firstOrigEdgeKey &&
		iter.GetOrigEdgeKeyLast() == lastOrigEdgeKey
}

func (c *EdgeBasedNodeContractor) resetEdgeCounters() {
	c.numShortcuts = 0
	c.numPrevEdges = 0
	c.numOrigEdges = 0
	c.numPrevOrigEdges = 0
	c.numAllEdges = 0
}

func (c *EdgeBasedNodeContractor) Close() {
	c.prepareGraph.Close()
	c.inEdgeExplorer = nil
	c.outEdgeExplorer = nil
	c.existingShortcutExplorer = nil
	c.sourceNodeOrigInEdgeExplorer = nil
	c.chBuilder = nil
	c.witnessPathSearcher.Close()
	c.sourceNodes = nil
	c.addedShortcuts = nil
	c.hierarchyDepths = nil
}

func (c *EdgeBasedNodeContractor) countShortcuts(edgeFrom, edgeTo *PrepareCHEntry, origEdgeCount int) {
	fromNode := edgeFrom.Parent.AdjNode
	toNode := edgeTo.AdjNode
	firstOrigEdgeKey := edgeFrom.FirstEdgeKey
	lastOrigEdgeKey := edgeTo.IncEdgeKey

	iter := c.existingShortcutExplorer.SetBaseNode(fromNode)
	for iter.Next() {
		if c.isSameShortcut(iter, toNode, firstOrigEdgeKey, lastOrigEdgeKey) {
			return
		}
	}

	for edgeTo != edgeFrom {
		c.numShortcuts++
		edgeTo = edgeTo.Parent
	}
	c.numOrigEdges += origEdgeCount
}

// GetNumPolledEdges returns the total number of polled edges for testing.
func (c *EdgeBasedNodeContractor) GetNumPolledEdges() int64 {
	return c.wpsStatsContr.NumPolls + c.wpsStatsHeur.NumPolls
}
