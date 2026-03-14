package ch

import (
	"fmt"
	"math"
	"sort"

	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// TurnCostFunction returns the turn weight for transitioning from inEdge to outEdge at viaNode.
type TurnCostFunction func(inEdge, viaNode, outEdge int) float64

// CHPreparationGraph is an adjacency-list graph optimized for CH contraction.
// Edges that are no longer needed (adjacent to contracted nodes) can be removed via Disconnect.
type CHPreparationGraph struct {
	nodes     int
	edges     int // number of original (non-shortcut) edges
	edgeBased bool
	turnCostFn TurnCostFunction

	prepareEdgesOut []prepareEdge
	prepareEdgesIn  []prepareEdge
	shortcutsByPrepareEdges []int
	degrees     []int
	origGraph   *origGraph
	origBuilder *origGraphBuilder
	nextShortcutID int
	ready       bool
}

func NewCHPreparationGraphNodeBased(nodes, edges int) *CHPreparationGraph {
	return newCHPreparationGraph(nodes, edges, false, func(_, _, _ int) float64 { return 0 })
}

func NewCHPreparationGraphEdgeBased(nodes, edges int, tcf TurnCostFunction) *CHPreparationGraph {
	return newCHPreparationGraph(nodes, edges, true, tcf)
}

func newCHPreparationGraph(nodes, edges int, edgeBased bool, tcf TurnCostFunction) *CHPreparationGraph {
	g := &CHPreparationGraph{
		nodes:           nodes,
		edges:           edges,
		edgeBased:       edgeBased,
		turnCostFn:      tcf,
		prepareEdgesOut: make([]prepareEdge, nodes),
		prepareEdgesIn:  make([]prepareEdge, nodes),
		degrees:         make([]int, nodes),
		nextShortcutID:  edges,
	}
	if edgeBased {
		g.origBuilder = newOrigGraphBuilder()
	}
	return g
}

func BuildFromGraph(pg *CHPreparationGraph, graph storage.Graph, w weighting.Weighting) {
	if graph.GetNodes() != pg.GetNodes() {
		panic(fmt.Sprintf("node count mismatch: %d vs %d", graph.GetNodes(), pg.GetNodes()))
	}
	if graph.GetEdges() != pg.GetOriginalEdges() {
		panic(fmt.Sprintf("edge count mismatch: %d vs %d", graph.GetEdges(), pg.GetOriginalEdges()))
	}
	iter := graph.GetAllEdges()
	for iter.Next() {
		weightFwd := w.CalcEdgeWeight(iter, false)
		weightBwd := w.CalcEdgeWeight(iter, true)
		pg.AddEdge(iter.GetBaseNode(), iter.GetAdjNode(), iter.GetEdge(), weightFwd, weightBwd)
	}
	pg.PrepareForContraction()
}

func BuildTurnCostFunctionFromWeighting(w weighting.Weighting) TurnCostFunction {
	return w.CalcTurnWeight
}

func (g *CHPreparationGraph) GetNodes() int         { return g.nodes }
func (g *CHPreparationGraph) GetOriginalEdges() int { return g.edges }
func (g *CHPreparationGraph) GetDegree(node int) int { return g.degrees[node] }

func (g *CHPreparationGraph) AddEdge(from, to, edge int, weightFwd, weightBwd float64) {
	g.checkNotReady()
	if from == to {
		panic("loop edges are not supported")
	}
	fwd := !math.IsInf(weightFwd, 0) && !math.IsNaN(weightFwd)
	bwd := !math.IsInf(weightBwd, 0) && !math.IsNaN(weightBwd)
	if !fwd && !bwd {
		return
	}
	pe := newPrepareBaseEdge(edge, from, to, float32(weightFwd), float32(weightBwd))
	if fwd {
		g.addOutEdge(from, pe)
		g.addInEdge(to, pe)
	}
	if bwd && from != to {
		g.addOutEdge(to, pe)
		g.addInEdge(from, pe)
	}
	if g.edgeBased {
		g.origBuilder.addEdge(from, to, edge, fwd, bwd)
	}
}

func (g *CHPreparationGraph) AddShortcut(from, to, origEdgeKeyFirst, origEdgeKeyLast, skipped1, skipped2 int, weight float64, origEdgeCount int) int {
	g.checkReady()
	var pe prepareEdge
	if g.edgeBased {
		pe = newEdgeBasedPrepareShortcut(g.nextShortcutID, from, to, origEdgeKeyFirst, origEdgeKeyLast, weight, skipped1, skipped2, origEdgeCount)
	} else {
		pe = newPrepareShortcut(g.nextShortcutID, from, to, weight, skipped1, skipped2, origEdgeCount)
	}
	g.addOutEdge(from, pe)
	if from != to {
		g.addInEdge(to, pe)
	}
	id := g.nextShortcutID
	g.nextShortcutID++
	return id
}

func (g *CHPreparationGraph) PrepareForContraction() {
	g.checkNotReady()
	if g.edgeBased {
		g.origGraph = g.origBuilder.build()
		g.origBuilder = nil
	}
	g.ready = true
}

func (g *CHPreparationGraph) SetShortcutForPrepareEdge(prepareEdge, shortcut int) {
	idx := prepareEdge - g.edges
	if needed := idx + 1 - len(g.shortcutsByPrepareEdges); needed > 0 {
		g.shortcutsByPrepareEdges = append(g.shortcutsByPrepareEdges, make([]int, needed)...)
	}
	g.shortcutsByPrepareEdges[idx] = shortcut
}

func (g *CHPreparationGraph) GetShortcutForPrepareEdge(prepareEdge int) int {
	if prepareEdge < g.edges {
		return prepareEdge
	}
	return g.shortcutsByPrepareEdges[prepareEdge-g.edges]
}

func (g *CHPreparationGraph) CreateOutEdgeExplorer() PrepareGraphEdgeExplorer {
	g.checkReady()
	return &prepareGraphEdgeExplorerImpl{prepareEdges: g.prepareEdgesOut, reverse: false}
}

func (g *CHPreparationGraph) CreateInEdgeExplorer() PrepareGraphEdgeExplorer {
	g.checkReady()
	return &prepareGraphEdgeExplorerImpl{prepareEdges: g.prepareEdgesIn, reverse: true}
}

func (g *CHPreparationGraph) CreateOutOrigEdgeExplorer() PrepareGraphOrigEdgeExplorer {
	g.checkReady()
	if !g.edgeBased {
		panic("orig out explorer is not available for node-based graph")
	}
	return g.origGraph.createOutOrigEdgeExplorer()
}

func (g *CHPreparationGraph) CreateInOrigEdgeExplorer() PrepareGraphOrigEdgeExplorer {
	g.checkReady()
	if !g.edgeBased {
		panic("orig in explorer is not available for node-based graph")
	}
	return g.origGraph.createInOrigEdgeExplorer()
}

func (g *CHPreparationGraph) GetTurnWeight(inEdgeKey, viaNode, outEdgeKey int) float64 {
	return g.turnCostFn(util.GetEdgeFromEdgeKey(inEdgeKey), viaNode, util.GetEdgeFromEdgeKey(outEdgeKey))
}

func (g *CHPreparationGraph) Disconnect(node int) []int {
	g.checkReady()
	neighborSet := make(map[int]struct{})

	currOut := g.prepareEdgesOut[node]
	for currOut != nil {
		adjNode := currOut.getNodeB()
		if adjNode == node {
			adjNode = currOut.getNodeA()
		}
		if adjNode == node {
			currOut = currOut.getNextOut(node)
			continue
		}
		g.removeInEdge(adjNode, currOut)
		neighborSet[adjNode] = struct{}{}
		currOut = currOut.getNextOut(node)
	}

	currIn := g.prepareEdgesIn[node]
	for currIn != nil {
		adjNode := currIn.getNodeB()
		if adjNode == node {
			adjNode = currIn.getNodeA()
		}
		if adjNode == node {
			currIn = currIn.getNextIn(node)
			continue
		}
		g.removeOutEdge(adjNode, currIn)
		neighborSet[adjNode] = struct{}{}
		currIn = currIn.getNextIn(node)
	}

	g.prepareEdgesOut[node] = nil
	g.prepareEdgesIn[node] = nil
	g.degrees[node] = 0

	neighbors := make([]int, 0, len(neighborSet))
	for n := range neighborSet {
		neighbors = append(neighbors, n)
	}
	sort.Ints(neighbors)
	return neighbors
}

func (g *CHPreparationGraph) Close() {
	g.checkReady()
	g.prepareEdgesOut = nil
	g.prepareEdgesIn = nil
	g.shortcutsByPrepareEdges = nil
	g.degrees = nil
	if g.edgeBased {
		g.origGraph = nil
	}
}

func (g *CHPreparationGraph) addOutEdge(node int, pe prepareEdge) {
	pe.setNextOut(node, g.prepareEdgesOut[node])
	g.prepareEdgesOut[node] = pe
	g.degrees[node]++
}

func (g *CHPreparationGraph) addInEdge(node int, pe prepareEdge) {
	pe.setNextIn(node, g.prepareEdgesIn[node])
	g.prepareEdgesIn[node] = pe
	g.degrees[node]++
}

func (g *CHPreparationGraph) removeOutEdge(node int, pe prepareEdge) {
	var prev prepareEdge
	curr := g.prepareEdgesOut[node]
	for curr != nil {
		if curr == pe {
			if prev == nil {
				g.prepareEdgesOut[node] = curr.getNextOut(node)
			} else {
				prev.setNextOut(node, curr.getNextOut(node))
			}
			g.degrees[node]--
		} else {
			prev = curr
		}
		curr = curr.getNextOut(node)
	}
}

func (g *CHPreparationGraph) removeInEdge(node int, pe prepareEdge) {
	var prev prepareEdge
	curr := g.prepareEdgesIn[node]
	for curr != nil {
		if curr == pe {
			if prev == nil {
				g.prepareEdgesIn[node] = curr.getNextIn(node)
			} else {
				prev.setNextIn(node, curr.getNextIn(node))
			}
			g.degrees[node]--
		} else {
			prev = curr
		}
		curr = curr.getNextIn(node)
	}
}

func (g *CHPreparationGraph) checkReady() {
	if !g.ready {
		panic("call PrepareForContraction() first")
	}
}

func (g *CHPreparationGraph) checkNotReady() {
	if g.ready {
		panic("cannot call this method after PrepareForContraction()")
	}
}

// --- prepareEdge interface and implementations ---

type prepareEdge interface {
	isShortcut() bool
	getPrepareEdge() int
	getNodeA() int
	getNodeB() int
	getWeightAB() float64
	getWeightBA() float64
	getOrigEdgeKeyFirstAB() int
	getOrigEdgeKeyFirstBA() int
	getOrigEdgeKeyLastAB() int
	getOrigEdgeKeyLastBA() int
	getSkipped1() int
	getSkipped2() int
	getOrigEdgeCount() int
	setSkipped1(int)
	setSkipped2(int)
	setWeight(float64)
	setOrigEdgeCount(int)
	getNextOut(base int) prepareEdge
	setNextOut(base int, pe prepareEdge)
	getNextIn(base int) prepareEdge
	setNextIn(base int, pe prepareEdge)
	String() string
}

// --- prepareBaseEdge ---

type prepareBaseEdge struct {
	edge     int
	nodeA    int
	nodeB    int
	weightAB float32
	weightBA float32
	nextOutA prepareEdge
	nextOutB prepareEdge
	nextInA  prepareEdge
	nextInB  prepareEdge
}

func newPrepareBaseEdge(edge, nodeA, nodeB int, weightAB, weightBA float32) *prepareBaseEdge {
	return &prepareBaseEdge{edge: edge, nodeA: nodeA, nodeB: nodeB, weightAB: weightAB, weightBA: weightBA}
}

func (e *prepareBaseEdge) isShortcut() bool     { return false }
func (e *prepareBaseEdge) getPrepareEdge() int  { return e.edge }
func (e *prepareBaseEdge) getNodeA() int        { return e.nodeA }
func (e *prepareBaseEdge) getNodeB() int        { return e.nodeB }
func (e *prepareBaseEdge) getWeightAB() float64 { return float64(e.weightAB) }
func (e *prepareBaseEdge) getWeightBA() float64 { return float64(e.weightBA) }

func (e *prepareBaseEdge) getOrigEdgeKeyFirstAB() int { return util.CreateEdgeKey(e.edge, false) }
func (e *prepareBaseEdge) getOrigEdgeKeyFirstBA() int { return util.CreateEdgeKey(e.edge, true) }
func (e *prepareBaseEdge) getOrigEdgeKeyLastAB() int  { return e.getOrigEdgeKeyFirstAB() }
func (e *prepareBaseEdge) getOrigEdgeKeyLastBA() int  { return e.getOrigEdgeKeyFirstBA() }

func (e *prepareBaseEdge) getSkipped1() int      { panic("not supported on base edge") }
func (e *prepareBaseEdge) getSkipped2() int      { panic("not supported on base edge") }
func (e *prepareBaseEdge) getOrigEdgeCount() int { return 1 }
func (e *prepareBaseEdge) setSkipped1(int)       { panic("not supported on base edge") }
func (e *prepareBaseEdge) setSkipped2(int)       { panic("not supported on base edge") }
func (e *prepareBaseEdge) setWeight(float64)     { panic("not supported on base edge") }
func (e *prepareBaseEdge) setOrigEdgeCount(int)  { panic("not supported on base edge") }

func (e *prepareBaseEdge) getNextOut(base int) prepareEdge {
	if base == e.nodeA {
		return e.nextOutA
	}
	if base == e.nodeB {
		return e.nextOutB
	}
	panic(fmt.Sprintf("base %d not adjacent to edge %d-%d", base, e.nodeA, e.nodeB))
}

func (e *prepareBaseEdge) setNextOut(base int, pe prepareEdge) {
	if base == e.nodeA {
		e.nextOutA = pe
	} else if base == e.nodeB {
		e.nextOutB = pe
	} else {
		panic(fmt.Sprintf("base %d not adjacent", base))
	}
}

func (e *prepareBaseEdge) getNextIn(base int) prepareEdge {
	if base == e.nodeA {
		return e.nextInA
	}
	if base == e.nodeB {
		return e.nextInB
	}
	panic(fmt.Sprintf("base %d not adjacent", base))
}

func (e *prepareBaseEdge) setNextIn(base int, pe prepareEdge) {
	if base == e.nodeA {
		e.nextInA = pe
	} else if base == e.nodeB {
		e.nextInB = pe
	} else {
		panic(fmt.Sprintf("base %d not adjacent", base))
	}
}

func (e *prepareBaseEdge) String() string {
	return fmt.Sprintf("%d-%d (%d) %v %v", e.nodeA, e.nodeB, e.edge, e.weightAB, e.weightBA)
}

// --- prepareShortcut (node-based) ---

type prepareShortcut struct {
	edge          int
	from          int
	to            int
	weight        float64
	skipped1      int
	skipped2      int
	origEdgeCount int
	nextOut       prepareEdge
	nextIn        prepareEdge
}

func newPrepareShortcut(edge, from, to int, weight float64, skipped1, skipped2, origEdgeCount int) *prepareShortcut {
	return &prepareShortcut{
		edge: edge, from: from, to: to, weight: weight,
		skipped1: skipped1, skipped2: skipped2, origEdgeCount: origEdgeCount,
	}
}

func (s *prepareShortcut) isShortcut() bool     { return true }
func (s *prepareShortcut) getPrepareEdge() int  { return s.edge }
func (s *prepareShortcut) getNodeA() int        { return s.from }
func (s *prepareShortcut) getNodeB() int        { return s.to }
func (s *prepareShortcut) getWeightAB() float64 { return s.weight }
func (s *prepareShortcut) getWeightBA() float64 { return s.weight }

func (s *prepareShortcut) getOrigEdgeKeyFirstAB() int { panic("not supported for node-based shortcut") }
func (s *prepareShortcut) getOrigEdgeKeyFirstBA() int { panic("not supported for node-based shortcut") }
func (s *prepareShortcut) getOrigEdgeKeyLastAB() int  { panic("not supported for node-based shortcut") }
func (s *prepareShortcut) getOrigEdgeKeyLastBA() int  { panic("not supported for node-based shortcut") }

func (s *prepareShortcut) getSkipped1() int      { return s.skipped1 }
func (s *prepareShortcut) getSkipped2() int      { return s.skipped2 }
func (s *prepareShortcut) getOrigEdgeCount() int { return s.origEdgeCount }
func (s *prepareShortcut) setSkipped1(v int)     { s.skipped1 = v }
func (s *prepareShortcut) setSkipped2(v int)     { s.skipped2 = v }
func (s *prepareShortcut) setWeight(v float64)   { s.weight = v }
func (s *prepareShortcut) setOrigEdgeCount(v int) { s.origEdgeCount = v }

func (s *prepareShortcut) getNextOut(_ int) prepareEdge    { return s.nextOut }
func (s *prepareShortcut) setNextOut(_ int, pe prepareEdge) { s.nextOut = pe }
func (s *prepareShortcut) getNextIn(_ int) prepareEdge     { return s.nextIn }
func (s *prepareShortcut) setNextIn(_ int, pe prepareEdge)  { s.nextIn = pe }

func (s *prepareShortcut) String() string {
	return fmt.Sprintf("%d-%d %v", s.from, s.to, s.weight)
}

// --- edgeBasedPrepareShortcut ---

type edgeBasedPrepareShortcut struct {
	prepareShortcut
	origEdgeKeyFirst int
	origEdgeKeyLast  int
}

func newEdgeBasedPrepareShortcut(edge, from, to, origEdgeKeyFirst, origEdgeKeyLast int, weight float64, skipped1, skipped2, origEdgeCount int) *edgeBasedPrepareShortcut {
	return &edgeBasedPrepareShortcut{
		prepareShortcut:  *newPrepareShortcut(edge, from, to, weight, skipped1, skipped2, origEdgeCount),
		origEdgeKeyFirst: origEdgeKeyFirst,
		origEdgeKeyLast:  origEdgeKeyLast,
	}
}

func (s *edgeBasedPrepareShortcut) getOrigEdgeKeyFirstAB() int { return s.origEdgeKeyFirst }
func (s *edgeBasedPrepareShortcut) getOrigEdgeKeyFirstBA() int { return s.origEdgeKeyFirst }
func (s *edgeBasedPrepareShortcut) getOrigEdgeKeyLastAB() int  { return s.origEdgeKeyLast }
func (s *edgeBasedPrepareShortcut) getOrigEdgeKeyLastBA() int  { return s.origEdgeKeyLast }

func (s *edgeBasedPrepareShortcut) String() string {
	return fmt.Sprintf("%d-%d (%d, %d) %v", s.from, s.to, s.origEdgeKeyFirst, s.origEdgeKeyLast, s.weight)
}

// --- prepareGraphEdgeExplorerImpl ---

type prepareGraphEdgeExplorerImpl struct {
	prepareEdges []prepareEdge
	reverse      bool
	node         int
	currEdge     prepareEdge
	nextEdge     prepareEdge
}

func (ex *prepareGraphEdgeExplorerImpl) SetBaseNode(node int) PrepareGraphEdgeIterator {
	ex.node = node
	ex.currEdge = nil
	ex.nextEdge = ex.prepareEdges[node]
	return ex
}

func (ex *prepareGraphEdgeExplorerImpl) Next() bool {
	ex.currEdge = ex.nextEdge
	if ex.currEdge == nil {
		return false
	}
	if ex.reverse {
		ex.nextEdge = ex.currEdge.getNextIn(ex.node)
	} else {
		ex.nextEdge = ex.currEdge.getNextOut(ex.node)
	}
	return true
}

func (ex *prepareGraphEdgeExplorerImpl) GetBaseNode() int { return ex.node }

func (ex *prepareGraphEdgeExplorerImpl) GetAdjNode() int {
	if ex.nodeAisBase() {
		return ex.currEdge.getNodeB()
	}
	return ex.currEdge.getNodeA()
}

func (ex *prepareGraphEdgeExplorerImpl) GetPrepareEdge() int { return ex.currEdge.getPrepareEdge() }
func (ex *prepareGraphEdgeExplorerImpl) IsShortcut() bool    { return ex.currEdge.isShortcut() }

func (ex *prepareGraphEdgeExplorerImpl) GetOrigEdgeKeyFirst() int {
	if ex.nodeAisBase() {
		return ex.currEdge.getOrigEdgeKeyFirstAB()
	}
	return ex.currEdge.getOrigEdgeKeyFirstBA()
}

func (ex *prepareGraphEdgeExplorerImpl) GetOrigEdgeKeyLast() int {
	if ex.nodeAisBase() {
		return ex.currEdge.getOrigEdgeKeyLastAB()
	}
	return ex.currEdge.getOrigEdgeKeyLastBA()
}

func (ex *prepareGraphEdgeExplorerImpl) GetSkipped1() int      { return ex.currEdge.getSkipped1() }
func (ex *prepareGraphEdgeExplorerImpl) GetSkipped2() int      { return ex.currEdge.getSkipped2() }
func (ex *prepareGraphEdgeExplorerImpl) GetOrigEdgeCount() int { return ex.currEdge.getOrigEdgeCount() }

func (ex *prepareGraphEdgeExplorerImpl) GetWeight() float64 {
	if ex.nodeAisBase() {
		if ex.reverse {
			return ex.currEdge.getWeightBA()
		}
		return ex.currEdge.getWeightAB()
	}
	if ex.reverse {
		return ex.currEdge.getWeightAB()
	}
	return ex.currEdge.getWeightBA()
}

func (ex *prepareGraphEdgeExplorerImpl) SetSkippedEdges(s1, s2 int) {
	ex.currEdge.setSkipped1(s1)
	ex.currEdge.setSkipped2(s2)
}

func (ex *prepareGraphEdgeExplorerImpl) SetWeight(w float64)      { ex.currEdge.setWeight(w) }
func (ex *prepareGraphEdgeExplorerImpl) SetOrigEdgeCount(c int) { ex.currEdge.setOrigEdgeCount(c) }

func (ex *prepareGraphEdgeExplorerImpl) String() string {
	if ex.currEdge == nil {
		return "not_started"
	}
	return ex.currEdge.String()
}

func (ex *prepareGraphEdgeExplorerImpl) nodeAisBase() bool {
	return ex.currEdge.getNodeA() == ex.node
}

// --- origGraph for edge-based CH ---

type origGraphEntry struct {
	adjNodeAndFwdFlag int
	keyAndBwdFlag     int
}

type origGraph struct {
	entries        []origGraphEntry
	firstEdgeByNode []int
}

func (og *origGraph) createOutOrigEdgeExplorer() PrepareGraphOrigEdgeExplorer {
	return &origEdgeIteratorImpl{graph: og, reverse: false}
}

func (og *origGraph) createInOrigEdgeExplorer() PrepareGraphOrigEdgeExplorer {
	return &origEdgeIteratorImpl{graph: og, reverse: true}
}

type origGraphBuilder struct {
	fromNodes         []int
	toNodesAndFwdFlags []int
	keysAndBwdFlags   []int
	maxFrom           int
}

func newOrigGraphBuilder() *origGraphBuilder {
	return &origGraphBuilder{maxFrom: -1}
}

func (b *origGraphBuilder) addEdge(from, to, edge int, fwd, bwd bool) {
	b.fromNodes = append(b.fromNodes, from)
	b.toNodesAndFwdFlags = append(b.toNodesAndFwdFlags, intWithFlag(to, fwd))
	b.keysAndBwdFlags = append(b.keysAndBwdFlags, intWithFlag(util.CreateEdgeKey(edge, false), bwd))
	b.maxFrom = max(b.maxFrom, from, to)

	b.fromNodes = append(b.fromNodes, to)
	b.toNodesAndFwdFlags = append(b.toNodesAndFwdFlags, intWithFlag(from, bwd))
	b.keysAndBwdFlags = append(b.keysAndBwdFlags, intWithFlag(util.CreateEdgeKey(edge, true), fwd))
}

func (b *origGraphBuilder) build() *origGraph {
	n := len(b.fromNodes)
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(i, j int) bool {
		return b.fromNodes[order[i]] < b.fromNodes[order[j]]
	})

	entries := make([]origGraphEntry, n)
	sortedFroms := make([]int, n)
	for i, idx := range order {
		sortedFroms[i] = b.fromNodes[idx]
		entries[i] = origGraphEntry{
			adjNodeAndFwdFlag: b.toNodesAndFwdFlags[idx],
			keyAndBwdFlag:     b.keysAndBwdFlags[idx],
		}
	}

	numFroms := b.maxFrom + 1
	firstEdgeByNode := make([]int, numFroms+1)
	edgeIdx := 0
	for from := 0; from < numFroms; from++ {
		for edgeIdx < n && sortedFroms[edgeIdx] < from {
			edgeIdx++
		}
		firstEdgeByNode[from] = edgeIdx
	}
	firstEdgeByNode[numFroms] = n

	return &origGraph{entries: entries, firstEdgeByNode: firstEdgeByNode}
}

func intWithFlag(val int, access bool) int {
	if val < 0 || val > math.MaxInt32 {
		panic(fmt.Sprintf("maximum node or edge key exceeded: %d, max: %d", val, math.MaxInt32))
	}
	val <<= 1
	if access {
		val++
	}
	return val
}

// --- origEdgeIteratorImpl ---

type origEdgeIteratorImpl struct {
	graph   *origGraph
	reverse bool
	node    int
	endEdge int
	index   int
}

func (it *origEdgeIteratorImpl) SetBaseNode(node int) PrepareGraphOrigEdgeIterator {
	it.node = node
	it.index = it.graph.firstEdgeByNode[node] - 1
	it.endEdge = it.graph.firstEdgeByNode[node+1]
	return it
}

func (it *origEdgeIteratorImpl) Next() bool {
	for {
		it.index++
		if it.index >= it.endEdge {
			return false
		}
		if it.hasAccess() {
			return true
		}
	}
}

func (it *origEdgeIteratorImpl) GetBaseNode() int { return it.node }

func (it *origEdgeIteratorImpl) GetAdjNode() int {
	return it.graph.entries[it.index].adjNodeAndFwdFlag >> 1
}

func (it *origEdgeIteratorImpl) GetOrigEdgeKeyFirst() int {
	return it.graph.entries[it.index].keyAndBwdFlag >> 1
}

func (it *origEdgeIteratorImpl) GetOrigEdgeKeyLast() int {
	return it.GetOrigEdgeKeyFirst()
}

func (it *origEdgeIteratorImpl) hasAccess() bool {
	entry := it.graph.entries[it.index]
	if it.reverse {
		return entry.keyAndBwdFlag&1 == 1
	}
	return entry.adjNodeAndFwdFlag&1 == 1
}

// ensure interfaces are satisfied
var (
	_ PrepareGraphEdgeExplorer    = (*prepareGraphEdgeExplorerImpl)(nil)
	_ PrepareGraphEdgeIterator    = (*prepareGraphEdgeExplorerImpl)(nil)
	_ PrepareGraphOrigEdgeExplorer = (*origEdgeIteratorImpl)(nil)
	_ PrepareGraphOrigEdgeIterator = (*origEdgeIteratorImpl)(nil)
)
