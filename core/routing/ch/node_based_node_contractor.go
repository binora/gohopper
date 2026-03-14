package ch

import (
	"fmt"
	"math"
	"time"

	"gohopper/core/storage"
	webapi "gohopper/web-api"
)

type nodeBasedParams struct {
	edgeDifferenceWeight     float32
	originalEdgesCountWeight float32
	maxPollFactorHeuristic   float64
	maxPollFactorContraction float64
}

type nodeBasedShortcut struct {
	prepareEdgeFwd int
	prepareEdgeBwd int
	from           int
	to             int
	skippedEdge1   int
	skippedEdge2   int
	weight         float64
	flags          int
}

func (sc *nodeBasedShortcut) String() string {
	dir := "->"
	if sc.flags == ScDirMask {
		dir = "<->"
	}
	return fmt.Sprintf("%d%s%d, weight:%v (%d,%d)", sc.from, dir, sc.to, sc.weight, sc.skippedEdge1, sc.skippedEdge2)
}

// prepareShortcutHandler is the callback for handling discovered shortcuts.
type prepareShortcutHandler func(fromNode, toNode int, weight float64, outgoingEdge, outOrigEdgeCount, incomingEdge, inOrigEdgeCount int)

// NodeBasedNodeContractor implements NodeContractor for node-based CH.
type NodeBasedNodeContractor struct {
	prepareGraph             *CHPreparationGraph
	params                   nodeBasedParams
	shortcuts                []nodeBasedShortcut
	chBuilder                *storage.CHStorageBuilder
	inEdgeExplorer           PrepareGraphEdgeExplorer
	outEdgeExplorer          PrepareGraphEdgeExplorer
	existingShortcutExplorer PrepareGraphEdgeExplorer
	witnessPathSearcher      *NodeBasedWitnessPathSearcher
	addedShortcutsCount      int
	dijkstraCount            int64
	dijkstraDuration         time.Duration
	meanDegree               float64
	originalEdgesCount       int
	shortcutsCount           int
}

func NewNodeBasedNodeContractor(prepareGraph *CHPreparationGraph, chBuilder *storage.CHStorageBuilder, pMap webapi.PMap) *NodeBasedNodeContractor {
	c := &NodeBasedNodeContractor{
		prepareGraph: prepareGraph,
		chBuilder:    chBuilder,
		params: nodeBasedParams{
			edgeDifferenceWeight:     10,
			originalEdgesCountWeight: 1,
			maxPollFactorHeuristic:   5,
			maxPollFactorContraction: 200,
		},
	}
	c.extractParams(pMap)
	return c
}

func (c *NodeBasedNodeContractor) extractParams(pMap webapi.PMap) {
	c.params.edgeDifferenceWeight = float32(pMap.GetFloat64(EdgeDifferenceWeight, float64(c.params.edgeDifferenceWeight)))
	c.params.originalEdgesCountWeight = float32(pMap.GetFloat64(OriginalEdgeCountWeight, float64(c.params.originalEdgesCountWeight)))
	c.params.maxPollFactorHeuristic = pMap.GetFloat64(MaxPollFactorHeuristicNode, c.params.maxPollFactorHeuristic)
	c.params.maxPollFactorContraction = pMap.GetFloat64(MaxPollFactorContractionNode, c.params.maxPollFactorContraction)
}

func (c *NodeBasedNodeContractor) InitFromGraph() {
	c.inEdgeExplorer = c.prepareGraph.CreateInEdgeExplorer()
	c.outEdgeExplorer = c.prepareGraph.CreateOutEdgeExplorer()
	c.existingShortcutExplorer = c.prepareGraph.CreateOutEdgeExplorer()
	c.witnessPathSearcher = NewNodeBasedWitnessPathSearcher(c.prepareGraph)
	c.meanDegree = float64(c.prepareGraph.GetOriginalEdges()) / float64(c.prepareGraph.GetNodes())
}

func (c *NodeBasedNodeContractor) Close() {
	c.prepareGraph.Close()
	c.shortcuts = nil
	c.chBuilder = nil
	c.inEdgeExplorer = nil
	c.outEdgeExplorer = nil
	c.existingShortcutExplorer = nil
	c.witnessPathSearcher = nil
}

func (c *NodeBasedNodeContractor) CalculatePriority(node int) float32 {
	c.shortcutsCount = 0
	c.originalEdgesCount = 0
	c.findAndHandleShortcuts(node, c.countShortcuts, int(c.meanDegree*c.params.maxPollFactorHeuristic))
	edgeDifference := c.shortcutsCount - c.prepareGraph.GetDegree(node)
	return c.params.edgeDifferenceWeight*float32(edgeDifference) +
		c.params.originalEdgesCountWeight*float32(c.originalEdgesCount)
}

func (c *NodeBasedNodeContractor) ContractNode(node int) []int {
	degree := c.findAndHandleShortcuts(node, c.addOrUpdateShortcut, int(c.meanDegree*c.params.maxPollFactorContraction))
	c.insertShortcuts(node)
	c.meanDegree = (c.meanDegree*2 + float64(degree)) / 3
	return c.prepareGraph.Disconnect(node)
}

func (c *NodeBasedNodeContractor) FinishContraction() {
	c.chBuilder.ReplaceSkippedEdges(c.prepareGraph.GetShortcutForPrepareEdge)
}

func (c *NodeBasedNodeContractor) GetAddedShortcutsCount() int64 {
	return int64(c.addedShortcutsCount)
}

func (c *NodeBasedNodeContractor) GetDijkstraSeconds() float32 {
	return float32(c.dijkstraDuration.Seconds())
}

func (c *NodeBasedNodeContractor) GetStatisticsString() string {
	return fmt.Sprintf("meanDegree: %.2f, dijkstras: %10d, mem: %10s",
		c.meanDegree, c.dijkstraCount, c.witnessPathSearcher.GetMemoryUsageAsString())
}

func (c *NodeBasedNodeContractor) insertShortcuts(node int) {
	c.shortcuts = c.shortcuts[:0]
	c.insertOutShortcuts(node)
	c.insertInShortcuts(node)
	origEdges := c.prepareGraph.GetOriginalEdges()
	for i := range c.shortcuts {
		sc := &c.shortcuts[i]
		shortcut := c.chBuilder.AddShortcutNodeBased(sc.from, sc.to, sc.flags, sc.weight, sc.skippedEdge1, sc.skippedEdge2)
		scEdge := origEdges + shortcut
		switch sc.flags {
		case ScFwdDir:
			c.prepareGraph.SetShortcutForPrepareEdge(sc.prepareEdgeFwd, scEdge)
		case ScBwdDir:
			c.prepareGraph.SetShortcutForPrepareEdge(sc.prepareEdgeBwd, scEdge)
		default:
			c.prepareGraph.SetShortcutForPrepareEdge(sc.prepareEdgeFwd, scEdge)
			c.prepareGraph.SetShortcutForPrepareEdge(sc.prepareEdgeBwd, scEdge)
		}
	}
	c.addedShortcutsCount += len(c.shortcuts)
}

func (c *NodeBasedNodeContractor) insertOutShortcuts(node int) {
	iter := c.outEdgeExplorer.SetBaseNode(node)
	for iter.Next() {
		if !iter.IsShortcut() {
			continue
		}
		c.shortcuts = append(c.shortcuts, nodeBasedShortcut{
			prepareEdgeFwd: iter.GetPrepareEdge(),
			prepareEdgeBwd: -1,
			from:           node,
			to:             iter.GetAdjNode(),
			skippedEdge1:   iter.GetSkipped1(),
			skippedEdge2:   iter.GetSkipped2(),
			flags:          ScFwdDir,
			weight:         iter.GetWeight(),
		})
	}
}

func (c *NodeBasedNodeContractor) insertInShortcuts(node int) {
	iter := c.inEdgeExplorer.SetBaseNode(node)
	for iter.Next() {
		if !iter.IsShortcut() {
			continue
		}
		skippedEdge1 := iter.GetSkipped2()
		skippedEdge2 := iter.GetSkipped1()
		bidir := false
		for i := range c.shortcuts {
			sc := &c.shortcuts[i]
			if sc.to == iter.GetAdjNode() &&
				math.Float64bits(sc.weight) == math.Float64bits(iter.GetWeight()) &&
				c.prepareGraph.GetShortcutForPrepareEdge(sc.skippedEdge1) == c.prepareGraph.GetShortcutForPrepareEdge(skippedEdge1) &&
				c.prepareGraph.GetShortcutForPrepareEdge(sc.skippedEdge2) == c.prepareGraph.GetShortcutForPrepareEdge(skippedEdge2) &&
				sc.flags == ScFwdDir {
				sc.flags = ScDirMask
				sc.prepareEdgeBwd = iter.GetPrepareEdge()
				bidir = true
				break
			}
		}
		if !bidir {
			c.shortcuts = append(c.shortcuts, nodeBasedShortcut{
				prepareEdgeFwd: -1,
				prepareEdgeBwd: iter.GetPrepareEdge(),
				from:           node,
				to:             iter.GetAdjNode(),
				skippedEdge1:   skippedEdge1,
				skippedEdge2:   skippedEdge2,
				flags:          ScBwdDir,
				weight:         iter.GetWeight(),
			})
		}
	}
}

func (c *NodeBasedNodeContractor) findAndHandleShortcuts(node int, handler prepareShortcutHandler, maxVisitedNodes int) int64 {
	var degree int64
	incomingEdges := c.inEdgeExplorer.SetBaseNode(node)
	for incomingEdges.Next() {
		fromNode := incomingEdges.GetAdjNode()
		if fromNode == node {
			panic(fmt.Sprintf("Unexpected loop-edge at node: %d", node))
		}
		incomingEdgeWeight := incomingEdges.GetWeight()
		if math.IsInf(incomingEdgeWeight, 1) {
			continue
		}
		outgoingEdges := c.outEdgeExplorer.SetBaseNode(node)
		c.witnessPathSearcher.Init(fromNode, node)
		degree++
		for outgoingEdges.Next() {
			toNode := outgoingEdges.GetAdjNode()
			if fromNode == toNode {
				continue
			}
			existingDirectWeight := incomingEdgeWeight + outgoingEdges.GetWeight()
			if math.IsInf(existingDirectWeight, 1) {
				continue
			}
			start := time.Now()
			c.dijkstraCount++
			maxWeight := c.witnessPathSearcher.FindUpperBound(toNode, existingDirectWeight, maxVisitedNodes)
			c.dijkstraDuration += time.Since(start)

			if maxWeight <= existingDirectWeight {
				continue
			}
			handler(fromNode, toNode, existingDirectWeight,
				outgoingEdges.GetPrepareEdge(), outgoingEdges.GetOrigEdgeCount(),
				incomingEdges.GetPrepareEdge(), incomingEdges.GetOrigEdgeCount())
		}
	}
	return degree
}

func (c *NodeBasedNodeContractor) countShortcuts(_, _ int, _ float64, _, outOrigEdgeCount, _, inOrigEdgeCount int) {
	c.shortcutsCount++
	c.originalEdgesCount += inOrigEdgeCount + outOrigEdgeCount
}

func (c *NodeBasedNodeContractor) addOrUpdateShortcut(fromNode, toNode int, weight float64, outgoingEdge, outOrigEdgeCount, incomingEdge, inOrigEdgeCount int) {
	iter := c.existingShortcutExplorer.SetBaseNode(fromNode)
	for iter.Next() {
		if iter.GetAdjNode() != toNode || !iter.IsShortcut() {
			continue
		}
		if weight < iter.GetWeight() {
			iter.SetWeight(weight)
			iter.SetSkippedEdges(incomingEdge, outgoingEdge)
			iter.SetOrigEdgeCount(inOrigEdgeCount + outOrigEdgeCount)
		}
		return
	}
	c.prepareGraph.AddShortcut(fromNode, toNode, -1, -1, incomingEdge, outgoingEdge, weight, inOrigEdgeCount+outOrigEdgeCount)
}
