package routing

import (
	"fmt"
	"time"

	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"
)

var _ BidirPathExtractor = (*DefaultBidirPathExtractor)(nil)

// DefaultBidirPathExtractor builds a Path from the two fwd- and bwd-shortest
// path tree entries of a bidirectional search.
type DefaultBidirPathExtractor struct {
	graph     storage.Graph
	weighting weighting.Weighting
	path      *Path
}

func ExtractBidirPath(graph storage.Graph, w weighting.Weighting, fwdEntry, bwdEntry *SPTEntry, weight float64) *Path {
	return NewDefaultBidirPathExtractor(graph, w).Extract(fwdEntry, bwdEntry, weight)
}

func NewDefaultBidirPathExtractor(graph storage.Graph, w weighting.Weighting) *DefaultBidirPathExtractor {
	return &DefaultBidirPathExtractor{
		graph:     graph,
		weighting: w,
		path:      NewPath(graph),
	}
}

func (e *DefaultBidirPathExtractor) Extract(fwdEntry, bwdEntry *SPTEntry, weight float64) *Path {
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

func (e *DefaultBidirPathExtractor) extractFwdPath(sptEntry *SPTEntry) {
	fwdRoot := e.followParentsUntilRoot(sptEntry, false)
	e.path.SetFromNode(fwdRoot.AdjNode)
	e.path.ReverseEdgeIDs()
}

func (e *DefaultBidirPathExtractor) extractBwdPath(sptEntry *SPTEntry) {
	bwdRoot := e.followParentsUntilRoot(sptEntry, true)
	e.path.SetEndNode(bwdRoot.AdjNode)
}

func (e *DefaultBidirPathExtractor) processMeetingPoint(fwdEntry, bwdEntry *SPTEntry) {
	inEdge := fwdEntry.Edge
	outEdge := bwdEntry.Edge
	if !util.EdgeIsValid(inEdge) || !util.EdgeIsValid(outEdge) {
		return
	}
	e.path.AddTime(e.weighting.CalcTurnMillis(inEdge, fwdEntry.AdjNode, outEdge))
}

func (e *DefaultBidirPathExtractor) followParentsUntilRoot(sptEntry *SPTEntry, reverse bool) *SPTEntry {
	currEntry := sptEntry
	parentEntry := currEntry.Parent
	for util.EdgeIsValid(currEntry.Edge) {
		e.onEdge(currEntry.Edge, currEntry.AdjNode, reverse, parentEntry.Edge)
		currEntry = parentEntry
		parentEntry = currEntry.Parent
	}
	return currEntry
}

func (e *DefaultBidirPathExtractor) onEdge(edge, adjNode int, reverse bool, prevOrNextEdge int) {
	edgeState := e.graph.GetEdgeIteratorState(edge, adjNode)
	e.path.AddDistance(edgeState.GetDistance())
	e.path.AddTime(calcMillisWithTurnMillis(e.weighting, edgeState, reverse, prevOrNextEdge))
	e.path.AddEdge(edge)
}
