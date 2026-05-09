package ch

import (
	"fmt"
	"time"

	"gohopper/core/routing"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"
)

var _ routing.BidirPathExtractor = (*EdgeBasedCHBidirPathExtractor)(nil)

// EdgeBasedCHBidirPathExtractor expands edge-based CH shortcuts and includes
// turn costs while reconstructing the base graph path.
type EdgeBasedCHBidirPathExtractor struct {
	weighting        weighting.Weighting
	path             *routing.Path
	shortcutUnpacker *ShortcutUnpacker
}

func NewEdgeBasedCHBidirPathExtractor(routingGraph storage.RoutingCHGraph) *EdgeBasedCHBidirPathExtractor {
	w, ok := routingGraph.GetWeighting().(weighting.Weighting)
	if !ok {
		panic(fmt.Sprintf("CH weighting %T does not implement weighting.Weighting", routingGraph.GetWeighting()))
	}
	extractor := &EdgeBasedCHBidirPathExtractor{
		weighting: w,
		path:      routing.NewPath(routingGraph.GetBaseGraph()),
	}
	extractor.shortcutUnpacker = NewShortcutUnpacker(routingGraph, edgeBasedCHPathVisitor{extractor: extractor}, true)
	return extractor
}

func (e *EdgeBasedCHBidirPathExtractor) Extract(fwdEntry, bwdEntry *routing.SPTEntry, weight float64) *routing.Path {
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

func (e *EdgeBasedCHBidirPathExtractor) extractFwdPath(sptEntry *routing.SPTEntry) {
	fwdRoot := e.followParentsUntilRoot(sptEntry, false)
	e.path.SetFromNode(fwdRoot.AdjNode)
	e.path.ReverseEdgeIDs()
}

func (e *EdgeBasedCHBidirPathExtractor) extractBwdPath(sptEntry *routing.SPTEntry) {
	bwdRoot := e.followParentsUntilRoot(sptEntry, true)
	e.path.SetEndNode(bwdRoot.AdjNode)
}

func (e *EdgeBasedCHBidirPathExtractor) processMeetingPoint(fwdEntry, bwdEntry *routing.SPTEntry) {
	if !util.EdgeIsValid(fwdEntry.IncEdge) || !util.EdgeIsValid(bwdEntry.IncEdge) {
		return
	}
	e.path.AddTime(e.weighting.CalcTurnMillis(fwdEntry.IncEdge, fwdEntry.AdjNode, bwdEntry.IncEdge))
}

func (e *EdgeBasedCHBidirPathExtractor) followParentsUntilRoot(sptEntry *routing.SPTEntry, reverse bool) *routing.SPTEntry {
	for currEntry := sptEntry; currEntry != nil; currEntry = currEntry.Parent {
		if !util.EdgeIsValid(currEntry.Edge) {
			return currEntry
		}
		prevOrNextEdge := util.NoEdge
		if currEntry.Parent != nil {
			prevOrNextEdge = currEntry.Parent.IncEdge
		}
		e.onEdge(currEntry.Edge, currEntry.AdjNode, reverse, prevOrNextEdge)
	}
	panic("edge-based CH path entry chain has no root")
}

func (e *EdgeBasedCHBidirPathExtractor) onEdge(edge, adjNode int, reverse bool, prevOrNextEdge int) {
	if reverse {
		e.shortcutUnpacker.VisitOriginalEdgesBwd(edge, adjNode, true, prevOrNextEdge)
		return
	}
	e.shortcutUnpacker.VisitOriginalEdgesFwd(edge, adjNode, true, prevOrNextEdge)
}

type edgeBasedCHPathVisitor struct {
	extractor *EdgeBasedCHBidirPathExtractor
}

func (v edgeBasedCHPathVisitor) Visit(edge util.EdgeIteratorState, reverse bool, prevOrNextEdgeID int) {
	extractor := v.extractor
	extractor.path.AddDistance(edge.GetDistance())
	extractor.path.AddTime(routing.CalcMillisWithTurnMillis(extractor.weighting, edge, reverse, prevOrNextEdgeID))
	extractor.path.AddEdge(edge.GetEdge())
}
