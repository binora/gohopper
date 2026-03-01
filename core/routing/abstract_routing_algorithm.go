package routing

import (
	"math"
	"time"

	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// AbstractRoutingAlgorithm provides shared state and helper methods for
// routing algorithm implementations. Go has no abstract classes, so concrete
// algorithms embed this struct and delegate to its methods.
type AbstractRoutingAlgorithm struct {
	Graph            storage.Graph
	Weighting        weighting.Weighting
	TraversalMode    routingutil.TraversalMode
	NodeAccess       storage.NodeAccess
	EdgeExplorer     util.EdgeExplorer
	MaxVisitedNodes  int
	TimeoutMillis    int64
	FinishTimeMillis int64
	AlreadyRun       bool
	VisitedNodes     int
}

// NewAbstractRoutingAlgorithm creates the shared base for a routing algorithm.
// It panics if the weighting supports turn costs but the traversal mode is not edge-based.
func NewAbstractRoutingAlgorithm(graph storage.Graph, w weighting.Weighting, mode routingutil.TraversalMode) AbstractRoutingAlgorithm {
	if w.HasTurnCosts() && !mode.IsEdgeBased() {
		panic("Weightings supporting turn costs cannot be used with node-based traversal mode")
	}
	return AbstractRoutingAlgorithm{
		Graph:            graph,
		Weighting:        w,
		TraversalMode:    mode,
		NodeAccess:       graph.GetNodeAccess(),
		EdgeExplorer:     graph.CreateEdgeExplorer(routingutil.AllEdges),
		MaxVisitedNodes:  math.MaxInt,
		TimeoutMillis:    math.MaxInt64,
		FinishTimeMillis: math.MaxInt64,
	}
}

// SetMaxVisitedNodes limits the search to the given number of nodes.
func (a *AbstractRoutingAlgorithm) SetMaxVisitedNodes(numberOfNodes int) {
	a.MaxVisitedNodes = numberOfNodes
}

// SetTimeoutMillis limits the search to the given time in milliseconds.
func (a *AbstractRoutingAlgorithm) SetTimeoutMillis(timeoutMillis int64) {
	a.TimeoutMillis = timeoutMillis
}

// Accept determines whether an edge should be accepted during traversal.
// For edge-based traversal, u-turn decisions are deferred to calcTurnWeight.
// For node-based traversal, u-turns (same edge as previous) are rejected for performance.
func (a *AbstractRoutingAlgorithm) Accept(iter util.EdgeIteratorState, prevOrNextEdgeID int) bool {
	return a.TraversalMode.IsEdgeBased() || iter.GetEdge() != prevOrNextEdgeID
}

// CheckAlreadyRun ensures the algorithm is only used once. It panics if
// called more than once.
func (a *AbstractRoutingAlgorithm) CheckAlreadyRun() {
	if a.AlreadyRun {
		panic("Create a new instance per call")
	}
	a.AlreadyRun = true
}

// SetupFinishTime calculates the absolute finish time from the configured timeout.
// If the addition overflows, FinishTimeMillis remains math.MaxInt64.
func (a *AbstractRoutingAlgorithm) SetupFinishTime() {
	now := time.Now().UnixMilli()
	finish := now + a.TimeoutMillis
	// Detect overflow: if TimeoutMillis is positive and finish wrapped around.
	if a.TimeoutMillis > 0 && finish < now {
		a.FinishTimeMillis = math.MaxInt64
	} else {
		a.FinishTimeMillis = finish
	}
}

// IsMaxVisitedNodesExceeded reports whether the visited node count has
// exceeded the configured limit.
func (a *AbstractRoutingAlgorithm) IsMaxVisitedNodesExceeded() bool {
	return a.MaxVisitedNodes < a.VisitedNodes
}

// IsTimeoutExceeded reports whether the current time has exceeded the finish time.
func (a *AbstractRoutingAlgorithm) IsTimeoutExceeded() bool {
	return a.FinishTimeMillis < math.MaxInt64 && time.Now().UnixMilli() > a.FinishTimeMillis
}

// CreateEmptyPath returns a new empty Path backed by the algorithm's graph.
func (a *AbstractRoutingAlgorithm) CreateEmptyPath() *Path {
	return NewPath(a.Graph)
}

// GetVisitedNodes returns the number of nodes visited during the search.
func (a *AbstractRoutingAlgorithm) GetVisitedNodes() int {
	return a.VisitedNodes
}

// DefaultCalcPaths is a package-level helper that wraps CalcPath into a
// single-element slice, matching the default behaviour of calcPaths in Java.
func DefaultCalcPaths(algo RoutingAlgorithm, from, to int) []*Path {
	return []*Path{algo.CalcPath(from, to)}
}
