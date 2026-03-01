package routing

import (
	"container/heap"
	"math"
	"time"

	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// AbstractBidirAlgo implements bidirectional Dijkstra search. It merges the
// Java AbstractBidirAlgo and AbstractNonCHBidirAlgo into a single Go struct.
// Concrete algorithms (e.g. DijkstraBidirectionRef) embed this struct and
// provide a name and optional hook overrides.
type AbstractBidirAlgo struct {
	Graph         storage.Graph
	Weighting     weighting.Weighting
	TraversalMode routingutil.TraversalMode
	NodeAccess    storage.NodeAccess
	EdgeExplorer  util.EdgeExplorer

	from       int
	to         int
	fromOutEdge int
	toInEdge   int

	bestWeightMapFrom  map[int]*SPTEntry
	bestWeightMapTo    map[int]*SPTEntry
	bestWeightMapOther map[int]*SPTEntry

	pqOpenSetFrom sptEntryHeap
	pqOpenSetTo   sptEntryHeap

	currFrom *SPTEntry
	currTo   *SPTEntry

	BestFwdEntry *SPTEntry
	BestBwdEntry *SPTEntry
	BestWeight   float64

	maxVisitedNodes  int
	timeoutMillis    int64
	finishTimeMillis int64

	updateBestPath bool
	finishedFrom   bool
	finishedTo     bool

	visitedCountFrom int
	visitedCountTo   int

	alreadyRun bool

	additionalEdgeFilter routingutil.EdgeFilter

	// Name is set by the concrete algorithm.
	Name string

	// CreatePathExtractorFn allows concrete types to override path extraction.
	CreatePathExtractorFn func(graph storage.Graph, w weighting.Weighting) BidirPathExtractor

	// Hooks for algorithm-specific overrides (e.g., AStarBidirection).
	// When nil, default behavior is used.
	PreInitFn          func(from int, fromWeight float64, to int, toWeight float64) // called before init
	FinishedFn         func() bool                                                   // overrides finished()
	CreateStartEntryFn func(node int, weight float64, reverse bool) *SPTEntry        // overrides createStartEntry
	CreateEntryFn      func(edge util.EdgeIteratorState, weight float64, parent *SPTEntry, reverse bool) *SPTEntry
}

func NewAbstractBidirAlgo(graph storage.Graph, w weighting.Weighting, tMode routingutil.TraversalMode) AbstractBidirAlgo {
	if w.HasTurnCosts() && !tMode.IsEdgeBased() {
		panic("Weightings supporting turn costs cannot be used with node-based traversal mode")
	}
	size := min(max(200, graph.GetNodes()/10), 150_000)
	a := AbstractBidirAlgo{
		Graph:         graph,
		Weighting:     w,
		TraversalMode: tMode,
		NodeAccess:    graph.GetNodeAccess(),
		EdgeExplorer:  graph.CreateEdgeExplorer(routingutil.AllEdges),

		fromOutEdge: util.AnyEdge,
		toInEdge:    util.AnyEdge,

		BestWeight: math.MaxFloat64,

		maxVisitedNodes:  math.MaxInt,
		timeoutMillis:    math.MaxInt64,
		finishTimeMillis: math.MaxInt64,

		updateBestPath: true,

		pqOpenSetFrom:     make(sptEntryHeap, 0, size),
		pqOpenSetTo:       make(sptEntryHeap, 0, size),
		bestWeightMapFrom: make(map[int]*SPTEntry, size),
		bestWeightMapTo:   make(map[int]*SPTEntry, size),
	}
	return a
}

func (a *AbstractBidirAlgo) CalcPath(from, to int) *Path {
	return a.CalcPathEdgeToEdge(from, to, util.AnyEdge, util.AnyEdge)
}

func (a *AbstractBidirAlgo) CalcPathEdgeToEdge(from, to, fromOutEdge, toInEdge int) *Path {
	if (fromOutEdge != util.AnyEdge || toInEdge != util.AnyEdge) && !a.TraversalMode.IsEdgeBased() {
		panic("Restricting the start/target edges is only possible for edge-based graph traversal")
	}
	a.fromOutEdge = fromOutEdge
	a.toInEdge = toInEdge
	a.checkAlreadyRun()
	a.setupFinishTime()
	a.init(from, 0, to, 0)
	a.runAlgo()
	return a.extractPath()
}

func (a *AbstractBidirAlgo) CalcPaths(from, to int) []*Path {
	return []*Path{a.CalcPath(from, to)}
}

func (a *AbstractBidirAlgo) init(from int, fromWeight float64, to int, toWeight float64) {
	if a.PreInitFn != nil {
		a.PreInitFn(from, fromWeight, to, toWeight)
	}
	a.initFrom(from, fromWeight)
	a.initTo(to, toWeight)
	a.postInit(from, to)
}

func (a *AbstractBidirAlgo) initFrom(from int, weight float64) {
	a.from = from
	a.currFrom = a.createStartEntry(from, weight, false)
	heap.Push(&a.pqOpenSetFrom, a.currFrom)
	if !a.TraversalMode.IsEdgeBased() {
		a.bestWeightMapFrom[from] = a.currFrom
	}
}

func (a *AbstractBidirAlgo) initTo(to int, weight float64) {
	a.to = to
	a.currTo = a.createStartEntry(to, weight, true)
	heap.Push(&a.pqOpenSetTo, a.currTo)
	if !a.TraversalMode.IsEdgeBased() {
		a.bestWeightMapTo[to] = a.currTo
	}
}

func (a *AbstractBidirAlgo) createStartEntry(node int, weight float64, reverse bool) *SPTEntry {
	if a.CreateStartEntryFn != nil {
		return a.CreateStartEntryFn(node, weight, reverse)
	}
	return NewSPTEntry(node, weight)
}

func (a *AbstractBidirAlgo) postInit(from, to int) {
	if !a.TraversalMode.IsEdgeBased() {
		if a.updateBestPath {
			a.bestWeightMapOther = a.bestWeightMapFrom
			a.updateBestPathEntry(math.Inf(1), a.currFrom, to, true)
		}
	} else if from == to && a.fromOutEdge == util.AnyEdge && a.toInEdge == util.AnyEdge {
		if a.currFrom.Weight != 0 || a.currTo.Weight != 0 {
			panic("If from=to, the starting weight must be zero for from and to")
		}
		a.BestFwdEntry = a.currFrom
		a.BestBwdEntry = a.currTo
		a.BestWeight = 0
		a.finishedFrom = true
		a.finishedTo = true
		return
	}
	a.postInitFrom()
	a.postInitTo()
}

func (a *AbstractBidirAlgo) postInitFrom() {
	if a.fromOutEdge == util.AnyEdge {
		a.fillEdgesFrom()
	} else {
		a.fillEdgesFromUsingFilter(func(edgeState util.EdgeIteratorState) bool {
			return edgeState.GetEdge() == a.fromOutEdge
		})
	}
}

func (a *AbstractBidirAlgo) postInitTo() {
	if a.toInEdge == util.AnyEdge {
		a.fillEdgesTo()
	} else {
		a.fillEdgesToUsingFilter(func(edgeState util.EdgeIteratorState) bool {
			return edgeState.GetEdge() == a.toInEdge
		})
	}
}

func (a *AbstractBidirAlgo) fillEdgesFromUsingFilter(filter routingutil.EdgeFilter) {
	a.additionalEdgeFilter = filter
	a.finishedFrom = !a.fillEdgesFrom()
	a.additionalEdgeFilter = nil
}

func (a *AbstractBidirAlgo) fillEdgesToUsingFilter(filter routingutil.EdgeFilter) {
	a.additionalEdgeFilter = filter
	a.finishedTo = !a.fillEdgesTo()
	a.additionalEdgeFilter = nil
}

func (a *AbstractBidirAlgo) runAlgo() {
	for !a.finished() && !a.isMaxVisitedNodesExceeded() && !a.isTimeoutExceeded() {
		if !a.finishedFrom {
			a.finishedFrom = !a.fillEdgesFrom()
		}
		if !a.finishedTo {
			a.finishedTo = !a.fillEdgesTo()
		}
	}
}

func (a *AbstractBidirAlgo) finished() bool {
	if a.FinishedFn != nil {
		return a.FinishedFn()
	}
	if a.finishedFrom || a.finishedTo {
		return true
	}
	return a.currFrom.Weight+a.currTo.Weight >= a.BestWeight
}

func (a *AbstractBidirAlgo) fillEdgesFrom() bool {
	for {
		if a.pqOpenSetFrom.Len() == 0 {
			return false
		}
		a.currFrom = heap.Pop(&a.pqOpenSetFrom).(*SPTEntry)
		if !a.currFrom.Deleted {
			break
		}
	}
	a.visitedCountFrom++
	a.bestWeightMapOther = a.bestWeightMapTo
	a.fillEdges(a.currFrom, &a.pqOpenSetFrom, a.bestWeightMapFrom, false)
	return true
}

func (a *AbstractBidirAlgo) fillEdgesTo() bool {
	for {
		if a.pqOpenSetTo.Len() == 0 {
			return false
		}
		a.currTo = heap.Pop(&a.pqOpenSetTo).(*SPTEntry)
		if !a.currTo.Deleted {
			break
		}
	}
	a.visitedCountTo++
	a.bestWeightMapOther = a.bestWeightMapFrom
	a.fillEdges(a.currTo, &a.pqOpenSetTo, a.bestWeightMapTo, true)
	return true
}

func (a *AbstractBidirAlgo) fillEdges(currEdge *SPTEntry, prioQueue *sptEntryHeap, bestWeightMap map[int]*SPTEntry, reverse bool) {
	iter := a.EdgeExplorer.SetBaseNode(currEdge.AdjNode)
	for iter.Next() {
		if !a.accept(iter, currEdge.Edge) {
			continue
		}

		weight := a.calcWeight(iter, currEdge, reverse)
		if math.IsInf(weight, 1) {
			continue
		}

		traversalID := a.TraversalMode.CreateTraversalID(iter, reverse)
		entry, exists := bestWeightMap[traversalID]
		if !exists {
			entry = a.createEntry(iter, weight, currEdge, reverse)
			bestWeightMap[traversalID] = entry
			heap.Push(prioQueue, entry)
		} else if entry.GetWeightOfVisitedPath() > weight {
			entry.Deleted = true
			isBestEntry := (reverse && entry == a.BestBwdEntry) || (!reverse && entry == a.BestFwdEntry)
			entry = a.createEntry(iter, weight, currEdge, reverse)
			bestWeightMap[traversalID] = entry
			heap.Push(prioQueue, entry)
			if isBestEntry {
				if reverse {
					a.BestBwdEntry = entry
				} else {
					a.BestFwdEntry = entry
				}
			}
		} else {
			continue
		}

		if a.updateBestPath {
			edgeWeight := math.Inf(1)
			if a.TraversalMode.IsEdgeBased() {
				edgeWeight = a.Weighting.CalcEdgeWeight(iter, reverse)
			}
			a.updateBestPathEntry(edgeWeight, entry, traversalID, reverse)
		}
	}
}

func (a *AbstractBidirAlgo) createEntry(edge util.EdgeIteratorState, weight float64, parent *SPTEntry, reverse bool) *SPTEntry {
	if a.CreateEntryFn != nil {
		return a.CreateEntryFn(edge, weight, parent, reverse)
	}
	return NewSPTEntryFull(edge.GetEdge(), edge.GetAdjNode(), weight, parent)
}

func (a *AbstractBidirAlgo) calcWeight(iter util.EdgeIteratorState, currEdge *SPTEntry, reverse bool) float64 {
	return CalcWeightWithTurnWeight(a.Weighting, iter, reverse, currEdge.Edge) + currEdge.GetWeightOfVisitedPath()
}

func (a *AbstractBidirAlgo) accept(iter util.EdgeIteratorState, prevOrNextEdgeID int) bool {
	if !a.TraversalMode.IsEdgeBased() && iter.GetEdge() == prevOrNextEdgeID {
		return false
	}
	return a.additionalEdgeFilter == nil || a.additionalEdgeFilter(iter)
}

func (a *AbstractBidirAlgo) updateBestPathEntry(edgeWeight float64, entry *SPTEntry, traversalID int, reverse bool) {
	entryOther, ok := a.bestWeightMapOther[traversalID]
	if !ok {
		return
	}

	weight := entry.GetWeightOfVisitedPath() + entryOther.GetWeightOfVisitedPath()
	if a.TraversalMode.IsEdgeBased() {
		if entryOther.Edge != entry.Edge {
			panic("cannot happen for edge based execution of " + a.Name)
		}
		entry = entry.Parent
		weight -= edgeWeight
	}

	if weight < a.BestWeight {
		if reverse {
			a.BestFwdEntry = entryOther
			a.BestBwdEntry = entry
		} else {
			a.BestFwdEntry = entry
			a.BestBwdEntry = entryOther
		}
		a.BestWeight = weight
	}
}

func (a *AbstractBidirAlgo) extractPath() *Path {
	if a.finished() {
		return a.createPathExtractor().Extract(a.BestFwdEntry, a.BestBwdEntry, a.BestWeight)
	}
	return NewPath(a.Graph)
}

func (a *AbstractBidirAlgo) createPathExtractor() BidirPathExtractor {
	if a.CreatePathExtractorFn != nil {
		return a.CreatePathExtractorFn(a.Graph, a.Weighting)
	}
	return NewDefaultBidirPathExtractor(a.Graph, a.Weighting)
}

func (a *AbstractBidirAlgo) GetVisitedNodes() int {
	return a.visitedCountFrom + a.visitedCountTo
}

func (a *AbstractBidirAlgo) GetName() string {
	return a.Name
}

func (a *AbstractBidirAlgo) SetMaxVisitedNodes(numberOfNodes int) {
	a.maxVisitedNodes = numberOfNodes
}

func (a *AbstractBidirAlgo) SetTimeoutMillis(timeoutMillis int64) {
	a.timeoutMillis = timeoutMillis
}

func (a *AbstractBidirAlgo) checkAlreadyRun() {
	if a.alreadyRun {
		panic("Create a new instance per call")
	}
	a.alreadyRun = true
}

func (a *AbstractBidirAlgo) setupFinishTime() {
	now := currentTimeMillis()
	finish := now + a.timeoutMillis
	if a.timeoutMillis > 0 && finish < now {
		// overflow: cap to max
		a.finishTimeMillis = math.MaxInt64
		return
	}
	a.finishTimeMillis = finish
}

func (a *AbstractBidirAlgo) isMaxVisitedNodesExceeded() bool {
	return a.maxVisitedNodes < a.GetVisitedNodes()
}

func (a *AbstractBidirAlgo) isTimeoutExceeded() bool {
	return a.finishTimeMillis < math.MaxInt64 && currentTimeMillis() > a.finishTimeMillis
}

func currentTimeMillis() int64 {
	return time.Now().UnixMilli()
}
