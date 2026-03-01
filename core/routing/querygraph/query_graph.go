package querygraph

import (
	"fmt"
	"math"
	"slices"

	routingutil "gohopper/core/routing/util"
	"gohopper/core/storage"
	"gohopper/core/storage/index"
	"gohopper/core/util"
)

var _ storage.Graph = (*QueryGraph)(nil)

type QueryGraph struct {
	baseGraph                  *storage.BaseGraph
	baseNodes                  int
	baseEdges                  int
	turnCostStorage            *storage.TurnCostStorage
	nodeAccess                 storage.NodeAccess
	queryOverlay               *QueryOverlay
	unfavoredEdges             []*VirtualEdgeIteratorState
	virtualEdgesAtRealNodes    map[int][]util.EdgeIteratorState
	virtualEdgesAtVirtualNodes [][]util.EdgeIteratorState
}

func Create(graph *storage.BaseGraph, snap *index.Snap) *QueryGraph {
	return CreateFromSnaps(graph, []*index.Snap{snap})
}

func CreateFromTwo(graph *storage.BaseGraph, fromSnap, toSnap *index.Snap) *QueryGraph {
	return CreateFromSnaps(graph.GetBaseGraph(), []*index.Snap{fromSnap, toSnap})
}

func CreateFromSnaps(graph *storage.BaseGraph, snaps []*index.Snap) *QueryGraph {
	return newQueryGraph(graph, snaps)
}

func newQueryGraph(graph *storage.BaseGraph, snaps []*index.Snap) *QueryGraph {
	qg := &QueryGraph{
		baseGraph: graph,
		baseNodes: graph.GetNodes(),
		baseEdges: graph.GetEdges(),
	}

	qg.queryOverlay = BuildQueryOverlay(graph, snaps)
	qg.nodeAccess = NewExtendedNodeAccess(graph.GetNodeAccess(), qg.queryOverlay.getVirtualNodes(), qg.baseNodes)
	qg.turnCostStorage = graph.GetTurnCostStorage()

	mainExplorer := graph.CreateEdgeExplorer(routingutil.AllEdges)
	qg.virtualEdgesAtRealNodes = qg.buildVirtualEdgesAtRealNodes(mainExplorer)
	qg.virtualEdgesAtVirtualNodes = qg.buildVirtualEdgesAtVirtualNodes()

	return qg
}

func (qg *QueryGraph) GetQueryOverlay() *QueryOverlay {
	return qg.queryOverlay
}

func (qg *QueryGraph) GetBaseGraph() *storage.BaseGraph {
	return qg.baseGraph
}

func (qg *QueryGraph) IsVirtualEdge(edgeID int) bool {
	return edgeID >= qg.baseEdges
}

func (qg *QueryGraph) IsVirtualNode(nodeID int) bool {
	return nodeID >= qg.baseNodes
}

func (qg *QueryGraph) UnfavorVirtualEdges(edgeIDs []int) {
	for _, id := range edgeIDs {
		qg.UnfavorVirtualEdge(id)
	}
}

func (qg *QueryGraph) UnfavorVirtualEdge(virtualEdgeID int) {
	if !qg.IsVirtualEdge(virtualEdgeID) {
		return
	}
	edge := qg.getVirtualEdge(qg.getInternalVirtualEdgeID(virtualEdgeID))
	edge.SetUnfavored(true)
	if !slices.Contains(qg.unfavoredEdges, edge) {
		qg.unfavoredEdges = append(qg.unfavoredEdges, edge)
	}

	reverseEdge := qg.getVirtualEdge(getPosOfReverseEdge(qg.getInternalVirtualEdgeID(virtualEdgeID)))
	reverseEdge.SetUnfavored(true)
	if !slices.Contains(qg.unfavoredEdges, reverseEdge) {
		qg.unfavoredEdges = append(qg.unfavoredEdges, reverseEdge)
	}
}

func (qg *QueryGraph) GetUnfavoredVirtualEdges() map[util.EdgeIteratorState]bool {
	result := make(map[util.EdgeIteratorState]bool, len(qg.unfavoredEdges))
	for _, e := range qg.unfavoredEdges {
		result[e] = true
	}
	return result
}

func (qg *QueryGraph) ClearUnfavoredStatus() {
	for _, edge := range qg.unfavoredEdges {
		edge.SetUnfavored(false)
	}
	qg.unfavoredEdges = qg.unfavoredEdges[:0]
}

func (qg *QueryGraph) GetNodes() int {
	return qg.queryOverlay.getVirtualNodes().Size() + qg.baseNodes
}

func (qg *QueryGraph) GetEdges() int {
	return qg.queryOverlay.getNumVirtualEdges()/2 + qg.baseEdges
}

func (qg *QueryGraph) GetNodeAccess() storage.NodeAccess {
	return qg.nodeAccess
}

func (qg *QueryGraph) GetBounds() util.BBox {
	return qg.baseGraph.GetBounds()
}

func (qg *QueryGraph) GetEdgeIteratorState(origEdgeID, adjNode int) util.EdgeIteratorState {
	if !qg.IsVirtualEdge(origEdgeID) {
		return qg.baseGraph.GetEdgeIteratorState(origEdgeID, adjNode)
	}
	edgeID := qg.getInternalVirtualEdgeID(origEdgeID)
	eis := qg.getVirtualEdge(edgeID)
	if eis.GetAdjNode() == adjNode || adjNode == math.MinInt32 {
		return eis
	}
	edgeID = getPosOfReverseEdge(edgeID)
	eis2 := qg.getVirtualEdge(edgeID)
	if eis2.GetAdjNode() == adjNode {
		return eis2
	}
	panic(fmt.Sprintf("edge %d not found with adjNode: %d. found edges were: %v, %v", origEdgeID, adjNode, eis, eis2))
}

func (qg *QueryGraph) GetEdgeIteratorStateForKey(edgeKey int) util.EdgeIteratorState {
	edge := util.GetEdgeFromEdgeKey(edgeKey)
	if !qg.IsVirtualEdge(edge) {
		return qg.baseGraph.GetEdgeIteratorStateForKey(edgeKey)
	}
	return qg.getVirtualEdge(edgeKey - 2*qg.baseEdges)
}

func (qg *QueryGraph) getVirtualEdge(edgeID int) *VirtualEdgeIteratorState {
	return qg.queryOverlay.getVirtualEdge(edgeID)
}

func getPosOfReverseEdge(edgeID int) int {
	if edgeID%2 == 0 {
		return edgeID + 1
	}
	return edgeID - 1
}

func (qg *QueryGraph) getInternalVirtualEdgeID(origEdgeID int) int {
	return 2 * (origEdgeID - qg.baseEdges)
}

func (qg *QueryGraph) CreateEdgeExplorer(edgeFilter routingutil.EdgeFilter) util.EdgeExplorer {
	mainExplorer := qg.baseGraph.CreateEdgeExplorer(edgeFilter)
	virtualEdgeIterator := NewVirtualEdgeIterator(edgeFilter, nil)
	return &queryGraphEdgeExplorer{
		qg:                  qg,
		mainExplorer:        mainExplorer,
		virtualEdgeIterator: virtualEdgeIterator,
	}
}

type queryGraphEdgeExplorer struct {
	qg                  *QueryGraph
	mainExplorer        util.EdgeExplorer
	virtualEdgeIterator *VirtualEdgeIterator
}

func (e *queryGraphEdgeExplorer) SetBaseNode(baseNode int) util.EdgeIterator {
	if e.qg.IsVirtualNode(baseNode) {
		virtualEdges := e.qg.virtualEdgesAtVirtualNodes[baseNode-e.qg.baseNodes]
		return e.virtualEdgeIterator.Reset(virtualEdges)
	}
	virtualEdges, ok := e.qg.virtualEdgesAtRealNodes[baseNode]
	if !ok {
		return e.mainExplorer.SetBaseNode(baseNode)
	}
	return e.virtualEdgeIterator.Reset(virtualEdges)
}

func (qg *QueryGraph) buildVirtualEdgesAtRealNodes(mainExplorer util.EdgeExplorer) map[int][]util.EdgeIteratorState {
	result := make(map[int][]util.EdgeIteratorState, len(qg.queryOverlay.getEdgeChangesAtRealNodes()))
	for node, edgeChanges := range qg.queryOverlay.getEdgeChangesAtRealNodes() {
		virtualEdges := make([]util.EdgeIteratorState, len(edgeChanges.AdditionalEdges))
		copy(virtualEdges, edgeChanges.AdditionalEdges)

		mainIter := mainExplorer.SetBaseNode(node)
		for mainIter.Next() {
			if !slices.Contains(edgeChanges.RemovedEdges, mainIter.GetEdge()) {
				virtualEdges = append(virtualEdges, mainIter.Detach(false))
			}
		}
		result[node] = virtualEdges
	}
	return result
}

func (qg *QueryGraph) buildVirtualEdgesAtVirtualNodes() [][]util.EdgeIteratorState {
	result := make([][]util.EdgeIteratorState, qg.queryOverlay.getVirtualNodes().Size())
	for i := range qg.queryOverlay.getVirtualNodes().Size() {
		result[i] = []util.EdgeIteratorState{
			qg.queryOverlay.getVirtualEdge(i*4 + SnapBase),
			qg.queryOverlay.getVirtualEdge(i*4 + SnapAdj),
		}
	}
	return result
}

func (qg *QueryGraph) GetAllEdges() storage.AllEdgesIterator {
	panic("not supported yet")
}

func (qg *QueryGraph) Edge(_, _ int) util.EdgeIteratorState {
	panic("QueryGraph cannot be modified")
}

func (qg *QueryGraph) GetTurnCostStorage() *storage.TurnCostStorage {
	return qg.turnCostStorage
}

func (qg *QueryGraph) GetOtherNode(edge, node int) int {
	if qg.IsVirtualEdge(edge) {
		return qg.GetEdgeIteratorState(edge, node).GetBaseNode()
	}
	return qg.baseGraph.GetOtherNode(edge, node)
}

func (qg *QueryGraph) IsAdjacentToNode(edge, node int) bool {
	if qg.IsVirtualEdge(edge) {
		virtualEdge := qg.GetEdgeIteratorState(edge, node)
		return virtualEdge.GetBaseNode() == node || virtualEdge.GetAdjNode() == node
	}
	return qg.baseGraph.IsAdjacentToNode(edge, node)
}

func (qg *QueryGraph) GetVirtualEdges() []*VirtualEdgeIteratorState {
	return qg.queryOverlay.getVirtualEdges()
}

func (qg *QueryGraph) GetClosestEdges() []int {
	return qg.queryOverlay.getClosestEdges()
}
