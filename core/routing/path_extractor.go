package routing

import (
	"fmt"
	"math"
	"time"

	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// ExtractPath builds a Path from the SPTEntry chain produced by a routing algorithm.
func ExtractPath(graph storage.Graph, w weighting.Weighting, sptEntry *SPTEntry) *Path {
	pe := &pathExtractor{
		graph:    graph,
		weighting: w,
		path:     NewPath(graph),
	}
	return pe.extract(sptEntry)
}

type pathExtractor struct {
	graph    storage.Graph
	weighting weighting.Weighting
	path     *Path
}

func (pe *pathExtractor) extract(sptEntry *SPTEntry) *Path {
	if sptEntry == nil {
		return pe.path
	}
	start := time.Now()
	pe.extractPath(sptEntry)
	pe.path.SetFound(true)
	pe.path.SetWeight(sptEntry.Weight)
	pe.path.SetDebugInfo(fmt.Sprintf("path extraction: %d us", time.Since(start).Microseconds()))
	return pe.path
}

func (pe *pathExtractor) extractPath(sptEntry *SPTEntry) {
	root := pe.followParentsUntilRoot(sptEntry)
	pe.path.ReverseEdgeIDs()
	pe.path.SetFromNode(root.AdjNode)
	pe.path.SetEndNode(sptEntry.AdjNode)
}

func (pe *pathExtractor) followParentsUntilRoot(sptEntry *SPTEntry) *SPTEntry {
	currEntry := sptEntry
	parentEntry := currEntry.Parent
	for util.EdgeIsValid(currEntry.Edge) {
		pe.onEdge(currEntry.Edge, currEntry.AdjNode, parentEntry.Edge)
		currEntry = currEntry.Parent
		parentEntry = currEntry.Parent
	}
	return currEntry
}

func (pe *pathExtractor) onEdge(edge, adjNode, prevEdge int) {
	edgeState := pe.graph.GetEdgeIteratorState(edge, adjNode)
	pe.path.AddDistance(edgeState.GetDistance())
	pe.path.AddTime(CalcMillisWithTurnMillis(pe.weighting, edgeState, false, prevEdge))
	pe.path.AddEdge(edge)
}

// CalcMillisWithTurnMillis calculates the time in milliseconds to traverse an edge,
// including turn costs from the previous edge.
func CalcMillisWithTurnMillis(w weighting.Weighting, edgeState util.EdgeIteratorState, reverse bool, prevOrNextEdgeID int) int64 {
	edgeMillis := w.CalcEdgeMillis(edgeState, reverse)
	if edgeMillis == math.MaxInt64 {
		return edgeMillis
	}
	if !util.EdgeIsValid(prevOrNextEdgeID) {
		return edgeMillis
	}
	origEdgeID := edgeState.GetEdge()
	var turnMillis int64
	if reverse {
		turnMillis = w.CalcTurnMillis(origEdgeID, edgeState.GetBaseNode(), prevOrNextEdgeID)
	} else {
		turnMillis = w.CalcTurnMillis(prevOrNextEdgeID, edgeState.GetBaseNode(), origEdgeID)
	}
	if turnMillis == math.MaxInt64 {
		return turnMillis
	}
	return edgeMillis + turnMillis
}
