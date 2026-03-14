package ch

import (
	"fmt"

	"gohopper/core/storage"
	"gohopper/core/util"
)

// Visitor is called for each original (non-shortcut) edge encountered during unpacking.
type Visitor interface {
	Visit(edge util.EdgeIteratorState, reverse bool, prevOrNextEdgeId int)
}

// ShortcutUnpacker recursively unpacks CH shortcuts into their constituent original edges.
type ShortcutUnpacker struct {
	graph        storage.RoutingCHGraph
	visitor      Visitor
	edgeBased    bool
	reverseOrder bool
}

func NewShortcutUnpacker(graph storage.RoutingCHGraph, visitor Visitor, edgeBased bool) *ShortcutUnpacker {
	return &ShortcutUnpacker{graph: graph, visitor: visitor, edgeBased: edgeBased}
}

func (u *ShortcutUnpacker) VisitOriginalEdgesFwd(edgeId, adjNode int, reverseOrder bool, prevOrNextEdgeId int) {
	u.doVisitOriginalEdges(edgeId, adjNode, reverseOrder, false, prevOrNextEdgeId)
}

func (u *ShortcutUnpacker) VisitOriginalEdgesBwd(edgeId, adjNode int, reverseOrder bool, prevOrNextEdgeId int) {
	u.doVisitOriginalEdges(edgeId, adjNode, reverseOrder, true, prevOrNextEdgeId)
}

func (u *ShortcutUnpacker) doVisitOriginalEdges(edgeId, adjNode int, reverseOrder, reverse bool, prevOrNextEdgeId int) {
	u.reverseOrder = reverseOrder
	edge := u.graph.GetEdgeIteratorState(edgeId, adjNode)
	if edge == nil {
		panic(fmt.Sprintf("Edge with id: %d does not exist or does not touch node %d", edgeId, adjNode))
	}
	u.expandEdge(edge, reverse, prevOrNextEdgeId)
}

func (u *ShortcutUnpacker) expandEdge(edge storage.RoutingCHEdgeIteratorState, reverse bool, prevOrNextEdgeId int) {
	if !edge.IsShortcut() {
		u.visitor.Visit(u.graph.GetBaseGraph().GetEdgeIteratorState(edge.GetOrigEdge(), edge.GetAdjNode()), reverse, prevOrNextEdgeId)
		return
	}
	if u.edgeBased {
		u.expandSkippedEdgesEdgeBased(edge.GetSkippedEdge1(), edge.GetSkippedEdge2(), edge.GetBaseNode(), edge.GetAdjNode(), reverse, prevOrNextEdgeId)
	} else {
		u.expandSkippedEdgesNodeBased(edge.GetSkippedEdge1(), edge.GetSkippedEdge2(), edge.GetBaseNode(), edge.GetAdjNode(), reverse)
	}
}

func (u *ShortcutUnpacker) expandSkippedEdgesEdgeBased(skippedEdge1, skippedEdge2, base, adj int, reverse bool, prevOrNextEdgeId int) {
	if reverse {
		skippedEdge1, skippedEdge2 = skippedEdge2, skippedEdge1
	}
	sk2 := u.graph.GetEdgeIteratorState(skippedEdge2, adj)
	sk1 := u.graph.GetEdgeIteratorState(skippedEdge1, sk2.GetBaseNode())
	if base == adj && (sk1.GetAdjNode() == sk1.GetBaseNode() || sk2.GetAdjNode() == sk2.GetBaseNode()) {
		panic(fmt.Sprintf("detected edge where a skipped edge is a loop. base: %d, adj: %d, skip1: %d, skip2: %d, reverse: %v",
			base, adj, skippedEdge1, skippedEdge2, reverse))
	}
	adjEdge := u.getOppositeEdge(sk1, base)
	if u.reverseOrder {
		u.expandEdge(sk2, reverse, adjEdge)
		u.expandEdge(sk1, reverse, prevOrNextEdgeId)
	} else {
		u.expandEdge(sk1, reverse, prevOrNextEdgeId)
		u.expandEdge(sk2, reverse, adjEdge)
	}
}

func (u *ShortcutUnpacker) expandSkippedEdgesNodeBased(skippedEdge1, skippedEdge2, base, adj int, reverse bool) {
	sk2 := u.graph.GetEdgeIteratorState(skippedEdge2, adj)
	var sk1 storage.RoutingCHEdgeIteratorState
	if sk2 == nil {
		sk2 = u.graph.GetEdgeIteratorState(skippedEdge1, adj)
		sk1 = u.graph.GetEdgeIteratorState(skippedEdge2, sk2.GetBaseNode())
	} else {
		sk1 = u.graph.GetEdgeIteratorState(skippedEdge1, sk2.GetBaseNode())
	}
	if u.reverseOrder {
		u.expandEdge(sk2, reverse, util.NoEdge)
		u.expandEdge(sk1, reverse, util.NoEdge)
	} else {
		u.expandEdge(sk1, reverse, util.NoEdge)
		u.expandEdge(sk2, reverse, util.NoEdge)
	}
}

func (u *ShortcutUnpacker) getOppositeEdge(edgeState storage.RoutingCHEdgeIteratorState, adjNode int) int {
	adjacentToNode := u.graph.GetBaseGraph().IsAdjacentToNode(util.GetEdgeFromEdgeKey(edgeState.GetOrigEdgeKeyLast()), adjNode)
	if adjacentToNode {
		return util.GetEdgeFromEdgeKey(edgeState.GetOrigEdgeKeyFirst())
	}
	return util.GetEdgeFromEdgeKey(edgeState.GetOrigEdgeKeyLast())
}
