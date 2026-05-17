package routing

import (
	"container/heap"
	"fmt"
	"math"
	"time"

	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// AbstractBidirCHAlgo contains the shared logic for bidirectional CH queries.
// It searches the upward CH graph in both directions and lets concrete
// algorithms override entry creation and stall-on-demand behavior.
type AbstractBidirCHAlgo struct {
	Graph         storage.RoutingCHGraph
	TraversalMode routingutil.TraversalMode

	inEdgeExplorer  storage.RoutingCHEdgeExplorer
	outEdgeExplorer storage.RoutingCHEdgeExplorer
	innerExplorer   util.EdgeExplorer

	from        int
	to          int
	fromOutEdge int
	toInEdge    int

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

	levelEdgeFilter storage.CHEdgeFilter

	Name string

	CreatePathExtractorFn func() BidirPathExtractor
	PreInitFn             func(from int, fromWeight float64, to int, toWeight float64)
	CreateStartEntryFn    func(node int, weight float64, reverse bool) *SPTEntry
	CreateCHEntryFn       func(edge, adjNode, incEdge int, weight float64, parent *SPTEntry, reverse bool) *SPTEntry
	FromEntryCanBeSkipped func() bool
	ToEntryCanBeSkipped   func() bool
	// FinishedFn lets subclasses override the bidirectional termination
	// condition (Java pattern: AlternativeRouteCH overrides finished() to
	// extend the search past the first optimum). When nil the default is used.
	FinishedFn func() bool
}

func NewAbstractBidirCHAlgo(graph storage.RoutingCHGraph, tMode routingutil.TraversalMode) AbstractBidirCHAlgo {
	if graph.HasTurnCosts() && !tMode.IsEdgeBased() {
		panic("Weightings supporting turn costs cannot be used with node-based traversal mode")
	}
	size := min(max(graph.GetNodes()/10, 200), 150_000)
	return AbstractBidirCHAlgo{
		Graph:             graph,
		TraversalMode:     tMode,
		inEdgeExplorer:    graph.CreateInEdgeExplorer(),
		outEdgeExplorer:   graph.CreateOutEdgeExplorer(),
		innerExplorer:     graph.GetBaseGraph().CreateEdgeExplorer(routingutil.AllEdges),
		fromOutEdge:       util.AnyEdge,
		toInEdge:          util.AnyEdge,
		BestWeight:        math.MaxFloat64,
		maxVisitedNodes:   math.MaxInt,
		timeoutMillis:     math.MaxInt64,
		finishTimeMillis:  math.MaxInt64,
		updateBestPath:    true,
		levelEdgeFilter:   newCHLevelEdgeFilter(graph),
		pqOpenSetFrom:     make(sptEntryHeap, 0, size),
		pqOpenSetTo:       make(sptEntryHeap, 0, size),
		bestWeightMapFrom: make(map[int]*SPTEntry, size),
		bestWeightMapTo:   make(map[int]*SPTEntry, size),
	}
}

func (a *AbstractBidirCHAlgo) CalcPath(from, to int) *Path {
	return a.CalcPathEdgeToEdge(from, to, util.AnyEdge, util.AnyEdge)
}

func (a *AbstractBidirCHAlgo) CalcPathEdgeToEdge(from, to, fromOutEdge, toInEdge int) *Path {
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

func (a *AbstractBidirCHAlgo) CalcPaths(from, to int) []*Path {
	return []*Path{a.CalcPath(from, to)}
}

func (a *AbstractBidirCHAlgo) init(from int, fromWeight float64, to int, toWeight float64) {
	if a.PreInitFn != nil {
		a.PreInitFn(from, fromWeight, to, toWeight)
	}
	a.initFrom(from, fromWeight)
	a.initTo(to, toWeight)
	a.postInit(from, to)
}

func (a *AbstractBidirCHAlgo) initFrom(from int, weight float64) {
	a.from = from
	a.currFrom = a.createStartEntry(from, weight, false)
	heap.Push(&a.pqOpenSetFrom, a.currFrom)
	if !a.TraversalMode.IsEdgeBased() {
		a.bestWeightMapFrom[from] = a.currFrom
	}
}

func (a *AbstractBidirCHAlgo) initTo(to int, weight float64) {
	a.to = to
	a.currTo = a.createStartEntry(to, weight, true)
	heap.Push(&a.pqOpenSetTo, a.currTo)
	if !a.TraversalMode.IsEdgeBased() {
		a.bestWeightMapTo[to] = a.currTo
	}
}

func (a *AbstractBidirCHAlgo) postInit(from, to int) {
	if !a.TraversalMode.IsEdgeBased() {
		if a.updateBestPath {
			a.bestWeightMapOther = a.bestWeightMapFrom
			a.updateBestPathEntry(a.currFrom, util.NoEdge, to, true)
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

func (a *AbstractBidirCHAlgo) postInitFrom() {
	a.fillEdgesFromUsingFilter(a.initialEdgeFilter(a.fromOutEdge, true))
}

func (a *AbstractBidirCHAlgo) postInitTo() {
	a.fillEdgesToUsingFilter(a.initialEdgeFilter(a.toInEdge, false))
}

func (a *AbstractBidirCHAlgo) initialEdgeFilter(restrictedEdge int, firstOrigEdge bool) storage.CHEdgeFilter {
	if restrictedEdge == util.AnyEdge {
		if a.TraversalMode.IsEdgeBased() {
			return storage.AllCHEdges
		}
		return a.levelEdgeFilter
	}

	levelFilter := a.levelEdgeFilter
	if a.TraversalMode.IsEdgeBased() {
		levelFilter = nil
	}
	return func(edge storage.RoutingCHEdgeIteratorState) bool {
		if levelFilter != nil && !levelFilter(edge) {
			return false
		}
		return origEdgeID(edge, firstOrigEdge) == restrictedEdge
	}
}

func origEdgeID(edge storage.RoutingCHEdgeIteratorState, firstOrigEdge bool) int {
	if firstOrigEdge {
		return util.GetEdgeFromEdgeKey(edge.GetOrigEdgeKeyFirst())
	}
	return util.GetEdgeFromEdgeKey(edge.GetOrigEdgeKeyLast())
}

func (a *AbstractBidirCHAlgo) fillEdgesFromUsingFilter(edgeFilter storage.CHEdgeFilter) {
	a.finishedFrom = !a.withLevelEdgeFilter(edgeFilter, a.fillEdgesFrom)
}

func (a *AbstractBidirCHAlgo) fillEdgesToUsingFilter(edgeFilter storage.CHEdgeFilter) {
	a.finishedTo = !a.withLevelEdgeFilter(edgeFilter, a.fillEdgesTo)
}

func (a *AbstractBidirCHAlgo) withLevelEdgeFilter(edgeFilter storage.CHEdgeFilter, fillEdges func() bool) bool {
	prev := a.levelEdgeFilter
	a.levelEdgeFilter = edgeFilter
	defer func() { a.levelEdgeFilter = prev }()
	return fillEdges()
}

func (a *AbstractBidirCHAlgo) runAlgo() {
	for !a.finished() && !a.isMaxVisitedNodesExceeded() && !a.isTimeoutExceeded() {
		if !a.finishedFrom {
			a.finishedFrom = !a.fillEdgesFrom()
		}
		if !a.finishedTo {
			a.finishedTo = !a.fillEdgesTo()
		}
	}
}

func (a *AbstractBidirCHAlgo) finished() bool {
	if a.FinishedFn != nil {
		return a.FinishedFn()
	}
	return a.finishedFrom && a.finishedTo ||
		a.currFrom.Weight >= a.BestWeight && a.currTo.Weight >= a.BestWeight
}

func (a *AbstractBidirCHAlgo) fillEdgesFrom() bool {
	entry, ok := nextSPTEntry(&a.pqOpenSetFrom)
	if !ok {
		return false
	}
	a.currFrom = entry
	a.visitedCountFrom++
	if a.FromEntryCanBeSkipped != nil && a.FromEntryCanBeSkipped() {
		return true
	}
	a.bestWeightMapOther = a.bestWeightMapTo
	a.fillEdges(a.currFrom, &a.pqOpenSetFrom, a.bestWeightMapFrom, a.outEdgeExplorer, false)
	return true
}

func (a *AbstractBidirCHAlgo) fillEdgesTo() bool {
	entry, ok := nextSPTEntry(&a.pqOpenSetTo)
	if !ok {
		return false
	}
	a.currTo = entry
	a.visitedCountTo++
	if a.ToEntryCanBeSkipped != nil && a.ToEntryCanBeSkipped() {
		return true
	}
	a.bestWeightMapOther = a.bestWeightMapFrom
	a.fillEdges(a.currTo, &a.pqOpenSetTo, a.bestWeightMapTo, a.inEdgeExplorer, true)
	return true
}

func nextSPTEntry(queue *sptEntryHeap) (*SPTEntry, bool) {
	for queue.Len() > 0 {
		entry := heap.Pop(queue).(*SPTEntry)
		if !entry.Deleted {
			return entry, true
		}
	}
	return nil, false
}

func (a *AbstractBidirCHAlgo) fillEdges(currEdge *SPTEntry, prioQueue *sptEntryHeap, bestWeightMap map[int]*SPTEntry, explorer storage.RoutingCHEdgeExplorer, reverse bool) {
	iter := explorer.SetBaseNode(currEdge.AdjNode)
	for iter.Next() {
		if !a.accept(iter, currEdge, reverse) {
			continue
		}

		weight := a.calcWeight(iter, currEdge, reverse)
		if math.IsInf(weight, 1) {
			continue
		}

		traversalID := a.createTraversalID(iter, reverse)
		entry := bestWeightMap[traversalID]
		if entry != nil {
			if entry.GetWeightOfVisitedPath() <= weight {
				continue
			}
			entry.Deleted = true
		}

		newEntry := a.createEntry(iter, weight, currEdge, reverse)
		bestWeightMap[traversalID] = newEntry
		heap.Push(prioQueue, newEntry)
		a.replaceBestEntry(entry, newEntry, reverse)

		if a.updateBestPath {
			origEdgeID := a.getOrigEdgeID(iter, reverse)
			a.updateBestPathEntry(newEntry, origEdgeID, traversalID, reverse)
		}
	}
}

func (a *AbstractBidirCHAlgo) replaceBestEntry(oldEntry, newEntry *SPTEntry, reverse bool) {
	if oldEntry == nil {
		return
	}
	switch {
	case reverse && oldEntry == a.BestBwdEntry:
		a.BestBwdEntry = newEntry
	case !reverse && oldEntry == a.BestFwdEntry:
		a.BestFwdEntry = newEntry
	}
}

func (a *AbstractBidirCHAlgo) createStartEntry(node int, weight float64, reverse bool) *SPTEntry {
	if a.CreateStartEntryFn != nil {
		return a.CreateStartEntryFn(node, weight, reverse)
	}
	return NewSPTEntry(node, weight)
}

func (a *AbstractBidirCHAlgo) createEntry(edge storage.RoutingCHEdgeIteratorState, weight float64, parent *SPTEntry, reverse bool) *SPTEntry {
	incEdge := a.getOrigEdgeID(edge, reverse)
	if a.CreateCHEntryFn != nil {
		return a.CreateCHEntryFn(edge.GetEdge(), edge.GetAdjNode(), incEdge, weight, parent, reverse)
	}
	entry := NewSPTEntryFull(edge.GetEdge(), edge.GetAdjNode(), weight, parent)
	entry.IncEdge = incEdge
	return entry
}

func (a *AbstractBidirCHAlgo) calcWeight(iter storage.RoutingCHEdgeIteratorState, currEdge *SPTEntry, reverse bool) float64 {
	return a.calcEdgeWeight(iter, reverse, a.getIncomingEdge(currEdge)) + currEdge.GetWeightOfVisitedPath()
}

func (a *AbstractBidirCHAlgo) calcEdgeWeight(edge storage.RoutingCHEdgeIteratorState, reverse bool, prevOrNextEdgeID int) float64 {
	edgeWeight := edge.GetWeight(reverse)
	if !a.TraversalMode.IsEdgeBased() {
		return edgeWeight
	}
	origEdgeID := a.getTurnOrigEdgeID(edge, reverse)
	if reverse {
		return edgeWeight + a.Graph.GetTurnWeight(origEdgeID, edge.GetBaseNode(), prevOrNextEdgeID)
	}
	return edgeWeight + a.Graph.GetTurnWeight(prevOrNextEdgeID, edge.GetBaseNode(), origEdgeID)
}

func (a *AbstractBidirCHAlgo) accept(edge storage.RoutingCHEdgeIteratorState, currEdge *SPTEntry, _ bool) bool {
	if !a.TraversalMode.IsEdgeBased() && edge.GetEdge() == a.getIncomingEdge(currEdge) {
		return false
	}
	return a.levelEdgeFilter == nil || a.levelEdgeFilter(edge)
}

func (a *AbstractBidirCHAlgo) createTraversalID(edge storage.RoutingCHEdgeIteratorState, reverse bool) int {
	if !a.TraversalMode.IsEdgeBased() {
		return edge.GetAdjNode()
	}
	if reverse {
		return reverseTraversalID(edge)
	}
	return edge.GetOrigEdgeKeyLast()
}

func reverseTraversalID(edge storage.RoutingCHEdgeIteratorState) int {
	key := edge.GetOrigEdgeKeyFirst()
	if edge.IsShortcut() || edge.GetBaseNode() == edge.GetAdjNode() {
		return key
	}
	return util.ReverseEdgeKey(key)
}

func (a *AbstractBidirCHAlgo) getIncomingEdge(entry *SPTEntry) int {
	if entry == nil {
		return util.NoEdge
	}
	if a.TraversalMode.IsEdgeBased() {
		return entry.IncEdge
	}
	return entry.Edge
}

func (a *AbstractBidirCHAlgo) getOrigEdgeID(edge storage.RoutingCHEdgeIteratorState, reverse bool) int {
	if !a.TraversalMode.IsEdgeBased() {
		return edge.GetEdge()
	}
	return util.GetEdgeFromEdgeKey(a.getOrigEdgeKey(edge, reverse))
}

func (a *AbstractBidirCHAlgo) getTurnOrigEdgeID(edge storage.RoutingCHEdgeIteratorState, reverse bool) int {
	return util.GetEdgeFromEdgeKey(a.getOrigEdgeKey(edge, !reverse))
}

func (a *AbstractBidirCHAlgo) getOrigEdgeKey(edge storage.RoutingCHEdgeIteratorState, reverse bool) int {
	if reverse {
		return edge.GetOrigEdgeKeyFirst()
	}
	return edge.GetOrigEdgeKeyLast()
}

func (a *AbstractBidirCHAlgo) updateBestPathEntry(entry *SPTEntry, origEdgeID, traversalID int, reverse bool) {
	if a.TraversalMode.IsEdgeBased() {
		a.updateBestPathEdgeBased(entry, origEdgeID, reverse)
		return
	}

	entryOther, ok := a.bestWeightMapOther[traversalID]
	if !ok {
		return
	}

	weight := entry.GetWeightOfVisitedPath() + entryOther.GetWeightOfVisitedPath()
	if weight < a.BestWeight {
		a.updateBestEntries(entry, entryOther, weight, reverse)
	}
}

func (a *AbstractBidirCHAlgo) updateBestPathEdgeBased(entry *SPTEntry, origEdgeID int, reverse bool) {
	oppositeNode := a.to
	oppositeEdge := a.toInEdge
	if reverse {
		oppositeNode = a.from
		oppositeEdge = a.fromOutEdge
	}
	if entry.AdjNode == oppositeNode && (oppositeEdge == util.AnyEdge || origEdgeID == oppositeEdge) {
		weight := entry.GetWeightOfVisitedPath()
		if weight < a.BestWeight {
			if reverse {
				a.BestFwdEntry = NewSPTEntry(oppositeNode, 0)
				a.BestBwdEntry = entry
			} else {
				a.BestFwdEntry = entry
				a.BestBwdEntry = NewSPTEntry(oppositeNode, 0)
			}
			a.BestWeight = weight
			return
		}
	}

	iter := a.innerExplorer.SetBaseNode(entry.AdjNode)
	for iter.Next() {
		traversalID := a.TraversalMode.CreateTraversalID(iter, reverse)
		entryOther := a.bestWeightMapOther[traversalID]
		if entryOther == nil {
			continue
		}

		edgeID := iter.GetEdge()
		var turnWeight float64
		if reverse {
			turnWeight = a.Graph.GetTurnWeight(edgeID, iter.GetBaseNode(), origEdgeID)
		} else {
			turnWeight = a.Graph.GetTurnWeight(origEdgeID, iter.GetBaseNode(), edgeID)
		}
		weight := entry.GetWeightOfVisitedPath() + entryOther.GetWeightOfVisitedPath() + turnWeight
		if weight < a.BestWeight {
			a.updateBestEntries(entry, entryOther, weight, reverse)
		}
	}
}

func (a *AbstractBidirCHAlgo) updateBestEntries(entry, entryOther *SPTEntry, weight float64, reverse bool) {
	if reverse {
		a.BestFwdEntry = entryOther
		a.BestBwdEntry = entry
	} else {
		a.BestFwdEntry = entry
		a.BestBwdEntry = entryOther
	}
	a.BestWeight = weight
}

func (a *AbstractBidirCHAlgo) extractPath() *Path {
	if a.finished() {
		return a.createPathExtractor().Extract(a.BestFwdEntry, a.BestBwdEntry, a.BestWeight)
	}
	return NewPath(a.Graph.GetBaseGraph())
}

func (a *AbstractBidirCHAlgo) createPathExtractor() BidirPathExtractor {
	if a.CreatePathExtractorFn != nil {
		return a.CreatePathExtractorFn()
	}
	w, ok := a.Graph.GetWeighting().(weighting.Weighting)
	if !ok {
		panic(fmt.Sprintf("CH weighting %T does not implement weighting.Weighting", a.Graph.GetWeighting()))
	}
	return NewDefaultBidirPathExtractor(a.Graph.GetBaseGraph(), w)
}

func (a *AbstractBidirCHAlgo) SetPathExtractorSupplier(fn func() BidirPathExtractor) {
	a.CreatePathExtractorFn = fn
}

func (a *AbstractBidirCHAlgo) GetVisitedNodes() int {
	return a.visitedCountFrom + a.visitedCountTo
}

func (a *AbstractBidirCHAlgo) GetName() string {
	return a.Name
}

func (a *AbstractBidirCHAlgo) SetMaxVisitedNodes(numberOfNodes int) {
	a.maxVisitedNodes = numberOfNodes
}

func (a *AbstractBidirCHAlgo) SetTimeoutMillis(timeoutMillis int64) {
	a.timeoutMillis = timeoutMillis
}

func (a *AbstractBidirCHAlgo) checkAlreadyRun() {
	if a.alreadyRun {
		panic("Create a new instance per call")
	}
	a.alreadyRun = true
}

func (a *AbstractBidirCHAlgo) setupFinishTime() {
	now := time.Now().UnixMilli()
	finish := now + a.timeoutMillis
	if a.timeoutMillis > 0 && finish < now {
		a.finishTimeMillis = math.MaxInt64
		return
	}
	a.finishTimeMillis = finish
}

func (a *AbstractBidirCHAlgo) isMaxVisitedNodesExceeded() bool {
	return a.maxVisitedNodes < a.GetVisitedNodes()
}

func (a *AbstractBidirCHAlgo) isTimeoutExceeded() bool {
	return a.finishTimeMillis < math.MaxInt64 && time.Now().UnixMilli() > a.finishTimeMillis
}

func newCHLevelEdgeFilter(graph storage.RoutingCHGraph) storage.CHEdgeFilter {
	maxNodes := graph.GetBaseGraph().GetBaseGraph().GetNodes()
	return func(edge storage.RoutingCHEdgeIteratorState) bool {
		base := edge.GetBaseNode()
		adj := edge.GetAdjNode()
		if base >= maxNodes || adj >= maxNodes {
			return true
		}
		return graph.GetLevel(base) <= graph.GetLevel(adj)
	}
}
