package querygraph

import (
	"fmt"
	"math"
	"slices"

	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"
)

var _ storage.RoutingCHGraph = (*QueryRoutingCHGraph)(nil)

// QueryRoutingCHGraph wraps a RoutingCHGraph with a QueryGraph so that virtual
// nodes/edges (from snapped points) can be used during CH routing queries.
// Create a QueryGraph first; then wrap the underlying RoutingCHGraph with this.
type QueryRoutingCHGraph struct {
	routingCHGraph storage.RoutingCHGraph
	weighting      storage.CHWeighting
	queryOverlay   *QueryOverlay
	queryGraph     *QueryGraph
	// queryGraphWeighting wraps weighting with the QueryGraphWeighting so that
	// turn weights involving virtual edges are correctly delegated.
	queryGraphWeighting *weighting.QueryGraphWeighting
	nodes               int

	virtualOutEdgesAtRealNodes map[int][]storage.RoutingCHEdgeIteratorState
	virtualInEdgesAtRealNodes  map[int][]storage.RoutingCHEdgeIteratorState
	virtualEdgesAtVirtualNodes [][]storage.RoutingCHEdgeIteratorState
}

// NewQueryRoutingCHGraph wires the supplied QueryGraph on top of a CH-routing
// graph. It pre-computes the virtual-edge lookup tables that the explorers and
// iterator state methods consult.
func NewQueryRoutingCHGraph(routingCHGraph storage.RoutingCHGraph, queryGraph *QueryGraph) *QueryRoutingCHGraph {
	baseWeighting, ok := routingCHGraph.GetWeighting().(weighting.Weighting)
	if !ok {
		panic(fmt.Sprintf("CH weighting %T does not implement weighting.Weighting", routingCHGraph.GetWeighting()))
	}
	g := &QueryRoutingCHGraph{
		routingCHGraph:      routingCHGraph,
		weighting:           routingCHGraph.GetWeighting(),
		queryOverlay:        queryGraph.GetQueryOverlay(),
		queryGraph:          queryGraph,
		queryGraphWeighting: weighting.NewQueryGraphWeighting(queryGraph.GetBaseGraph(), baseWeighting, queryGraph.GetClosestEdges()),
		nodes:               queryGraph.GetNodes(),
	}
	g.virtualOutEdgesAtRealNodes = g.buildVirtualEdgesAtRealNodes(routingCHGraph.CreateOutEdgeExplorer())
	g.virtualInEdgesAtRealNodes = g.buildVirtualEdgesAtRealNodes(routingCHGraph.CreateInEdgeExplorer())
	g.virtualEdgesAtVirtualNodes = g.buildVirtualEdgesAtVirtualNodes()
	return g
}

func (g *QueryRoutingCHGraph) GetNodes() int {
	return g.nodes
}

func (g *QueryRoutingCHGraph) GetEdges() int {
	return g.routingCHGraph.GetEdges() + g.queryOverlay.getNumVirtualEdges()
}

func (g *QueryRoutingCHGraph) GetShortcuts() int {
	return g.routingCHGraph.GetShortcuts()
}

func (g *QueryRoutingCHGraph) CreateInEdgeExplorer() storage.RoutingCHEdgeExplorer {
	return g.createEdgeExplorer(g.routingCHGraph.CreateInEdgeExplorer(), g.virtualInEdgesAtRealNodes)
}

func (g *QueryRoutingCHGraph) CreateOutEdgeExplorer() storage.RoutingCHEdgeExplorer {
	return g.createEdgeExplorer(g.routingCHGraph.CreateOutEdgeExplorer(), g.virtualOutEdgesAtRealNodes)
}

func (g *QueryRoutingCHGraph) createEdgeExplorer(explorer storage.RoutingCHEdgeExplorer, virtualEdgesAtRealNodes map[int][]storage.RoutingCHEdgeIteratorState) storage.RoutingCHEdgeExplorer {
	return &virtualCHEdgeExplorer{
		queryRoutingCHGraph:     g,
		explorer:                explorer,
		virtualEdgesAtRealNodes: virtualEdgesAtRealNodes,
		iter:                    &virtualCHEdgeIterator{},
	}
}

func (g *QueryRoutingCHGraph) GetEdgeIteratorState(chEdge, adjNode int) storage.RoutingCHEdgeIteratorState {
	if !g.isVirtualEdge(chEdge) {
		return g.routingCHGraph.GetEdgeIteratorState(chEdge, adjNode)
	}
	// todo (parity with Java): possible optimization - instead of building a new
	// virtual edge object reuse the ones we already built for virtualEdgesAtReal/VirtualNodes.
	return g.buildVirtualCHEdgeStateFromVirtual(g.getVirtualEdgeState(chEdge, adjNode))
}

func (g *QueryRoutingCHGraph) GetLevel(node int) int {
	if g.isVirtualNode(node) {
		return math.MaxInt
	}
	return g.routingCHGraph.GetLevel(node)
}

func (g *QueryRoutingCHGraph) GetTurnWeight(inEdge, viaNode, outEdge int) float64 {
	if !g.routingCHGraph.HasTurnCosts() {
		// node-based algorithms might pass in ch edge ids here -> return 0
		return 0
	}
	return g.queryGraphWeighting.CalcTurnWeight(inEdge, viaNode, outEdge)
}

func (g *QueryRoutingCHGraph) GetBaseGraph() storage.Graph {
	return g.queryGraph
}

func (g *QueryRoutingCHGraph) HasTurnCosts() bool {
	return g.routingCHGraph.HasTurnCosts()
}

func (g *QueryRoutingCHGraph) IsEdgeBased() bool {
	return g.routingCHGraph.IsEdgeBased()
}

func (g *QueryRoutingCHGraph) GetWeighting() storage.CHWeighting {
	return g.weighting
}

func (g *QueryRoutingCHGraph) Close() {
	g.routingCHGraph.Close()
	g.virtualEdgesAtVirtualNodes = nil
	g.virtualInEdgesAtRealNodes = nil
	g.virtualOutEdgesAtRealNodes = nil
}

func (g *QueryRoutingCHGraph) getVirtualEdgeState(virtualEdgeID, adjNode int) *VirtualEdgeIteratorState {
	if !g.isVirtualEdge(virtualEdgeID) {
		panic(fmt.Sprintf("not a virtual edge: %d", virtualEdgeID))
	}
	internalVirtualEdgeID := g.getInternalVirtualEdgeID(virtualEdgeID)
	virtualEdge := g.queryOverlay.getVirtualEdge(internalVirtualEdgeID)
	if virtualEdge.GetAdjNode() == adjNode || adjNode == math.MinInt32 {
		return virtualEdge
	}
	internalVirtualEdgeID = getPosOfReverseEdge(internalVirtualEdgeID)
	virtualEdge = g.queryOverlay.getVirtualEdge(internalVirtualEdgeID)
	if virtualEdge.GetAdjNode() != adjNode {
		panic(fmt.Sprintf("The virtual edge with ID %d does not touch node %d", virtualEdgeID, adjNode))
	}
	return virtualEdge
}

func (g *QueryRoutingCHGraph) buildVirtualEdgesAtRealNodes(explorer storage.RoutingCHEdgeExplorer) map[int][]storage.RoutingCHEdgeIteratorState {
	edgeChanges := g.queryOverlay.getEdgeChangesAtRealNodes()
	result := make(map[int][]storage.RoutingCHEdgeIteratorState, len(edgeChanges))
	for node, changes := range edgeChanges {
		virtualEdges := make([]storage.RoutingCHEdgeIteratorState, 0, len(changes.AdditionalEdges))
		for _, v := range changes.AdditionalEdges {
			if v.GetBaseNode() != node {
				panic(fmt.Sprintf("expected base node %d, got %d", node, v.GetBaseNode()))
			}
			edge := v.GetEdge()
			if g.queryGraph.IsVirtualEdge(edge) {
				edge = g.shiftVirtualEdgeIDForCH(edge)
			}
			virtualEdges = append(virtualEdges, g.buildVirtualCHEdgeState(v, edge))
		}
		iter := explorer.SetBaseNode(node)
		for iter.Next() {
			// shortcuts cannot be in the removed edge set because this was determined on the (base) query graph
			if iter.IsShortcut() {
				virtualEdges = append(virtualEdges, &virtualCHEdgeIteratorState{
					edge:             iter.GetEdge(),
					origEdge:         util.NoEdge,
					baseNode:         iter.GetBaseNode(),
					adjNode:          iter.GetAdjNode(),
					origEdgeKeyFirst: iter.GetOrigEdgeKeyFirst(),
					origEdgeKeyLast:  iter.GetOrigEdgeKeyLast(),
					skippedEdge1:     iter.GetSkippedEdge1(),
					skippedEdge2:     iter.GetSkippedEdge2(),
					weightFwd:        iter.GetWeight(false),
					weightBwd:        iter.GetWeight(true),
				})
			} else if !containsInt(changes.RemovedEdges, iter.GetOrigEdge()) {
				virtualEdges = append(virtualEdges, &virtualCHEdgeIteratorState{
					edge:             iter.GetEdge(),
					origEdge:         iter.GetOrigEdge(),
					baseNode:         iter.GetBaseNode(),
					adjNode:          iter.GetAdjNode(),
					origEdgeKeyFirst: iter.GetOrigEdgeKeyFirst(),
					origEdgeKeyLast:  iter.GetOrigEdgeKeyLast(),
					skippedEdge1:     util.NoEdge,
					skippedEdge2:     util.NoEdge,
					weightFwd:        iter.GetWeight(false),
					weightBwd:        iter.GetWeight(true),
				})
			}
		}
		result[node] = virtualEdges
	}
	return result
}

func (g *QueryRoutingCHGraph) buildVirtualEdgesAtVirtualNodes() [][]storage.RoutingCHEdgeIteratorState {
	virtualNodes := g.queryOverlay.getVirtualNodes().Size()
	result := make([][]storage.RoutingCHEdgeIteratorState, virtualNodes)
	for i := range virtualNodes {
		result[i] = []storage.RoutingCHEdgeIteratorState{
			g.buildVirtualCHEdgeStateFromVirtual(g.queryOverlay.getVirtualEdge(i*4 + SnapBase)),
			g.buildVirtualCHEdgeStateFromVirtual(g.queryOverlay.getVirtualEdge(i*4 + SnapAdj)),
		}
	}
	return result
}

func (g *QueryRoutingCHGraph) buildVirtualCHEdgeStateFromVirtual(virtualEdgeState *VirtualEdgeIteratorState) *virtualCHEdgeIteratorState {
	virtualCHEdge := g.shiftVirtualEdgeIDForCH(virtualEdgeState.GetEdge())
	return g.buildVirtualCHEdgeState(virtualEdgeState, virtualCHEdge)
}

func (g *QueryRoutingCHGraph) buildVirtualCHEdgeState(edgeState util.EdgeIteratorState, edgeID int) *virtualCHEdgeIteratorState {
	fwdWeight := g.weighting.CalcEdgeWeight(edgeState, false)
	bwdWeight := g.weighting.CalcEdgeWeight(edgeState, true)
	return &virtualCHEdgeIteratorState{
		edge:             edgeID,
		origEdge:         edgeState.GetEdge(),
		baseNode:         edgeState.GetBaseNode(),
		adjNode:          edgeState.GetAdjNode(),
		origEdgeKeyFirst: edgeState.GetEdgeKey(),
		origEdgeKeyLast:  edgeState.GetEdgeKey(),
		skippedEdge1:     util.NoEdge,
		skippedEdge2:     util.NoEdge,
		weightFwd:        fwdWeight,
		weightBwd:        bwdWeight,
	}
}

func (g *QueryRoutingCHGraph) shiftVirtualEdgeIDForCH(edge int) int {
	return edge + g.routingCHGraph.GetEdges() - g.routingCHGraph.GetBaseGraph().GetEdges()
}

func (g *QueryRoutingCHGraph) getInternalVirtualEdgeID(edge int) int {
	return 2 * (edge - g.routingCHGraph.GetEdges())
}

func (g *QueryRoutingCHGraph) isVirtualNode(node int) bool {
	return node >= g.routingCHGraph.GetNodes()
}

func (g *QueryRoutingCHGraph) isVirtualEdge(edge int) bool {
	return edge >= g.routingCHGraph.GetEdges()
}

func containsInt(s []int, target int) bool {
	return slices.Contains(s, target)
}

// virtualCHEdgeExplorer routes setBaseNode calls either to the wrapped CH
// explorer (for real nodes with no virtual changes) or to the precomputed
// virtual-edge slices for real nodes touched by virtual edges and for virtual
// nodes.
type virtualCHEdgeExplorer struct {
	queryRoutingCHGraph     *QueryRoutingCHGraph
	explorer                storage.RoutingCHEdgeExplorer
	virtualEdgesAtRealNodes map[int][]storage.RoutingCHEdgeIteratorState
	iter                    *virtualCHEdgeIterator
}

func (e *virtualCHEdgeExplorer) SetBaseNode(baseNode int) storage.RoutingCHEdgeIterator {
	if e.queryRoutingCHGraph.isVirtualNode(baseNode) {
		virtualEdges := e.queryRoutingCHGraph.virtualEdgesAtVirtualNodes[baseNode-e.queryRoutingCHGraph.routingCHGraph.GetNodes()]
		e.iter.reset(virtualEdges)
		return e.iter
	}
	virtualEdges, ok := e.virtualEdgesAtRealNodes[baseNode]
	if !ok {
		return e.explorer.SetBaseNode(baseNode)
	}
	e.iter.reset(virtualEdges)
	return e.iter
}

// virtualCHEdgeIteratorState is the immutable CH-edge view of a virtual edge or
// a copy of a real edge / shortcut we materialised during construction.
type virtualCHEdgeIteratorState struct {
	edge             int
	origEdge         int
	baseNode         int
	adjNode          int
	origEdgeKeyFirst int
	origEdgeKeyLast  int
	skippedEdge1     int
	skippedEdge2     int
	weightFwd        float64
	weightBwd        float64
}

func (s *virtualCHEdgeIteratorState) GetEdge() int             { return s.edge }
func (s *virtualCHEdgeIteratorState) GetOrigEdge() int         { return s.origEdge }
func (s *virtualCHEdgeIteratorState) GetOrigEdgeKeyFirst() int { return s.origEdgeKeyFirst }
func (s *virtualCHEdgeIteratorState) GetOrigEdgeKeyLast() int  { return s.origEdgeKeyLast }
func (s *virtualCHEdgeIteratorState) GetBaseNode() int         { return s.baseNode }
func (s *virtualCHEdgeIteratorState) GetAdjNode() int          { return s.adjNode }
func (s *virtualCHEdgeIteratorState) IsShortcut() bool         { return s.origEdge == util.NoEdge }

func (s *virtualCHEdgeIteratorState) GetSkippedEdge1() int { return s.skippedEdge1 }
func (s *virtualCHEdgeIteratorState) GetSkippedEdge2() int { return s.skippedEdge2 }

func (s *virtualCHEdgeIteratorState) GetWeight(reverse bool) float64 {
	if reverse {
		return s.weightBwd
	}
	return s.weightFwd
}

func (s *virtualCHEdgeIteratorState) String() string {
	return fmt.Sprintf("virtual: %d: %d->%d, orig: %d, weightFwd: %.2f, weightBwd: %.2f",
		s.edge, s.baseNode, s.adjNode, s.origEdge, s.weightFwd, s.weightBwd)
}

// virtualCHEdgeIterator iterates over a slice of precomputed CH edge states.
type virtualCHEdgeIterator struct {
	edges   []storage.RoutingCHEdgeIteratorState
	current int
}

func (it *virtualCHEdgeIterator) reset(edges []storage.RoutingCHEdgeIteratorState) {
	it.edges = edges
	it.current = -1
}

func (it *virtualCHEdgeIterator) Next() bool {
	it.current++
	return it.current < len(it.edges)
}

func (it *virtualCHEdgeIterator) current0() storage.RoutingCHEdgeIteratorState {
	return it.edges[it.current]
}

func (it *virtualCHEdgeIterator) GetEdge() int             { return it.current0().GetEdge() }
func (it *virtualCHEdgeIterator) GetOrigEdge() int         { return it.current0().GetOrigEdge() }
func (it *virtualCHEdgeIterator) GetOrigEdgeKeyFirst() int { return it.current0().GetOrigEdgeKeyFirst() }
func (it *virtualCHEdgeIterator) GetOrigEdgeKeyLast() int  { return it.current0().GetOrigEdgeKeyLast() }
func (it *virtualCHEdgeIterator) GetBaseNode() int         { return it.current0().GetBaseNode() }
func (it *virtualCHEdgeIterator) GetAdjNode() int          { return it.current0().GetAdjNode() }
func (it *virtualCHEdgeIterator) IsShortcut() bool         { return it.current0().IsShortcut() }

func (it *virtualCHEdgeIterator) GetSkippedEdge1() int {
	if !it.IsShortcut() {
		panic("Skipped edges are only available for shortcuts")
	}
	return it.current0().GetSkippedEdge1()
}

func (it *virtualCHEdgeIterator) GetSkippedEdge2() int {
	if !it.IsShortcut() {
		panic("Skipped edges are only available for shortcuts")
	}
	return it.current0().GetSkippedEdge2()
}

func (it *virtualCHEdgeIterator) GetWeight(reverse bool) float64 {
	return it.current0().GetWeight(reverse)
}

func (it *virtualCHEdgeIterator) String() string {
	if it.current < 0 {
		return "not started"
	}
	return fmt.Sprintf("%v, current: %d/%d", it.edges[it.current], it.current+1, len(it.edges))
}
