package ch

import (
	"fmt"
	"time"

	"gohopper/core/routing"
	"gohopper/core/storage"
	"gohopper/core/util"
)

var _ routing.BidirPathExtractor = (*NodeBasedCHBidirPathExtractor)(nil)

// NodeBasedCHBidirPathExtractor expands CH shortcuts while building a base
// graph path from bidirectional CH search entries.
type NodeBasedCHBidirPathExtractor struct {
	routingGraph     storage.RoutingCHGraph
	path             *routing.Path
	shortcutUnpacker *ShortcutUnpacker
}

func NewNodeBasedCHBidirPathExtractor(routingGraph storage.RoutingCHGraph) *NodeBasedCHBidirPathExtractor {
	e := &NodeBasedCHBidirPathExtractor{
		routingGraph: routingGraph,
		path:         routing.NewPath(routingGraph.GetBaseGraph()),
	}
	e.shortcutUnpacker = NewShortcutUnpacker(routingGraph, nodeBasedCHPathVisitor{extractor: e}, false)
	return e
}

func (e *NodeBasedCHBidirPathExtractor) Extract(fwdEntry, bwdEntry *routing.SPTEntry, weight float64) *routing.Path {
	if fwdEntry == nil || bwdEntry == nil {
		return e.path
	}
	if fwdEntry.AdjNode != bwdEntry.AdjNode {
		panic(fmt.Sprintf("forward and backward entries must have same adjacent nodes, fwdEntry:%s, bwdEntry:%s",
			fwdEntry, bwdEntry))
	}

	start := time.Now()
	e.extractFwdPath(fwdEntry)
	e.processMeetingPoint(fwdEntry, bwdEntry)
	e.extractBwdPath(bwdEntry)
	e.path.SetDebugInfo(fmt.Sprintf("path extraction: %d us", time.Since(start).Microseconds()))
	e.path.SetFound(true)
	e.path.SetWeight(weight)
	return e.path
}

func (e *NodeBasedCHBidirPathExtractor) extractFwdPath(sptEntry *routing.SPTEntry) {
	fwdRoot := e.followParentsUntilRoot(sptEntry, false)
	e.path.SetFromNode(fwdRoot.AdjNode)
	e.path.ReverseEdgeIDs()
}

func (e *NodeBasedCHBidirPathExtractor) extractBwdPath(sptEntry *routing.SPTEntry) {
	bwdRoot := e.followParentsUntilRoot(sptEntry, true)
	e.path.SetEndNode(bwdRoot.AdjNode)
}

func (e *NodeBasedCHBidirPathExtractor) processMeetingPoint(fwdEntry, bwdEntry *routing.SPTEntry) {
	inEdge := fwdEntry.Edge
	outEdge := bwdEntry.Edge
	if !util.EdgeIsValid(inEdge) || !util.EdgeIsValid(outEdge) {
		return
	}
	e.path.AddTime(e.routingGraph.GetWeighting().CalcTurnMillis(inEdge, fwdEntry.AdjNode, outEdge))
}

func (e *NodeBasedCHBidirPathExtractor) followParentsUntilRoot(sptEntry *routing.SPTEntry, reverse bool) *routing.SPTEntry {
	for currEntry := sptEntry; ; {
		if !util.EdgeIsValid(currEntry.Edge) {
			return currEntry
		}
		parentEntry := currEntry.Parent
		prevOrNextEdge := util.NoEdge
		if parentEntry != nil {
			prevOrNextEdge = parentEntry.Edge
		}
		e.onEdge(currEntry.Edge, currEntry.AdjNode, reverse, prevOrNextEdge)
		currEntry = parentEntry
	}
}

func (e *NodeBasedCHBidirPathExtractor) onEdge(edge, adjNode int, reverse bool, prevOrNextEdge int) {
	if reverse {
		e.shortcutUnpacker.VisitOriginalEdgesBwd(edge, adjNode, true, prevOrNextEdge)
		return
	}
	e.shortcutUnpacker.VisitOriginalEdgesFwd(edge, adjNode, true, prevOrNextEdge)
}

type nodeBasedCHPathVisitor struct {
	extractor *NodeBasedCHBidirPathExtractor
}

func (v nodeBasedCHPathVisitor) Visit(edge util.EdgeIteratorState, reverse bool, _ int) {
	extractor := v.extractor
	extractor.path.AddDistance(edge.GetDistance())
	extractor.path.AddTime(extractor.routingGraph.GetWeighting().CalcEdgeMillis(edge, reverse))
	extractor.path.AddEdge(edge.GetEdge())
}
