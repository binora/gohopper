package storage

import (
	"math"

	routingutil "gohopper/core/routing/util"
	"gohopper/core/util"
)

type routingCHEdgeIteratorImpl struct {
	routingCHEdgeIteratorStateImpl
	baseIterator *edgeIteratorImpl
	outgoing     bool
	incoming     bool
	nextEdgeID   int
}

func newRoutingCHEdgeIteratorOut(chStore *CHStorage, bg *BaseGraph, w CHWeighting) *routingCHEdgeIteratorImpl {
	return newRoutingCHEdgeIteratorImpl(chStore, bg, w, true, false)
}

func newRoutingCHEdgeIteratorIn(chStore *CHStorage, bg *BaseGraph, w CHWeighting) *routingCHEdgeIteratorImpl {
	return newRoutingCHEdgeIteratorImpl(chStore, bg, w, false, true)
}

func newRoutingCHEdgeIteratorImpl(chStore *CHStorage, bg *BaseGraph, w CHWeighting, outgoing, incoming bool) *routingCHEdgeIteratorImpl {
	baseIter := newEdgeIteratorImpl(bg, routingutil.AllEdges)
	it := &routingCHEdgeIteratorImpl{
		routingCHEdgeIteratorStateImpl: *newRoutingCHEdgeIteratorStateImpl(chStore, bg, &baseIter.EdgeIteratorStateImpl, w),
		baseIterator:                   baseIter,
		outgoing:                       outgoing,
		incoming:                       incoming,
	}
	return it
}

func (it *routingCHEdgeIteratorImpl) edgeState() util.EdgeIteratorState {
	return it.baseIterator
}

func (it *routingCHEdgeIteratorImpl) getOrigEdgeWeight(reverse bool) float64 {
	return it.weighting.CalcEdgeWeight(it.baseIterator, reverse)
}

func (it *routingCHEdgeIteratorImpl) GetWeight(reverse bool) float64 {
	if it.IsShortcut() {
		return it.store.GetWeight(it.shortcutPointer)
	}
	return it.getOrigEdgeWeight(reverse)
}

func (it *routingCHEdgeIteratorImpl) GetOrigEdge() int {
	if it.IsShortcut() {
		return util.NoEdge
	}
	return it.baseIterator.GetEdge()
}

func (it *routingCHEdgeIteratorImpl) GetOrigEdgeKeyFirst() int {
	if !it.IsShortcut() || !it.store.IsEdgeBased() {
		return it.baseIterator.GetEdgeKey()
	}
	return it.store.GetOrigEdgeKeyFirst(it.shortcutPointer)
}

func (it *routingCHEdgeIteratorImpl) GetOrigEdgeKeyLast() int {
	if !it.IsShortcut() || !it.store.IsEdgeBased() {
		return it.baseIterator.GetEdgeKey()
	}
	return it.store.GetOrigEdgeKeyLast(it.shortcutPointer)
}

func (it *routingCHEdgeIteratorImpl) GetBaseNode() int {
	if it.IsShortcut() {
		return it.baseNode
	}
	return it.baseIterator.GetBaseNode()
}

func (it *routingCHEdgeIteratorImpl) GetAdjNode() int {
	if it.IsShortcut() {
		return it.adjNode
	}
	return it.baseIterator.GetAdjNode()
}

func (it *routingCHEdgeIteratorImpl) SetBaseNode(baseNode int) RoutingCHEdgeIterator {
	it.baseIterator.SetBaseNode(baseNode)
	lastShortcut := it.store.GetLastShortcut(it.store.ToNodePointer(baseNode))
	if lastShortcut < 0 {
		it.nextEdgeID = it.baseIterator.EdgeID
	} else {
		it.nextEdgeID = it.baseGraph.GetEdges() + lastShortcut
	}
	it.edgeID = it.nextEdgeID
	return it
}

func (it *routingCHEdgeIteratorImpl) Next() bool {
	for it.nextEdgeID >= it.baseGraph.GetEdges() {
		it.shortcutPointer = it.store.ToShortcutPointer(it.nextEdgeID - it.baseGraph.GetEdges())
		it.baseNode = it.store.GetNodeA(it.shortcutPointer)
		it.adjNode = it.store.GetNodeB(it.shortcutPointer)
		it.edgeID = it.nextEdgeID
		it.nextEdgeID--
		if it.nextEdgeID < it.baseGraph.GetEdges() ||
			it.store.GetNodeA(it.store.ToShortcutPointer(it.nextEdgeID-it.baseGraph.GetEdges())) != it.baseNode {
			it.nextEdgeID = it.baseIterator.EdgeID
		}
		// Accept loop shortcuts in any direction, otherwise filter by direction
		if (it.baseNode == it.adjNode && (it.store.GetFwdAccess(it.shortcutPointer) || it.store.GetBwdAccess(it.shortcutPointer))) ||
			(it.outgoing && it.store.GetFwdAccess(it.shortcutPointer) || it.incoming && it.store.GetBwdAccess(it.shortcutPointer)) {
			return true
		}
	}

	for util.EdgeIsValid(it.baseIterator.nextEdgeID) {
		it.baseIterator.goToNext()
		it.edgeID = it.baseIterator.EdgeID
		if (it.outgoing && it.finiteWeight(false)) || (it.incoming && it.finiteWeight(true)) {
			return true
		}
	}
	return false
}

func (it *routingCHEdgeIteratorImpl) finiteWeight(reverse bool) bool {
	return !math.IsInf(it.getOrigEdgeWeight(reverse), 0)
}
