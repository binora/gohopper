package storage

import (
	"fmt"

	routingutil "gohopper/core/routing/util"
	"gohopper/core/util"
)

// edgeIteratorImpl iterates over edges of a single node, applying a filter.
// Implements both EdgeExplorer and EdgeIterator.
type edgeIteratorImpl struct {
	EdgeIteratorStateImpl
	filter     routingutil.EdgeFilter
	nextEdgeID int
}

func newEdgeIteratorImpl(bg *BaseGraph, filter routingutil.EdgeFilter) *edgeIteratorImpl {
	if filter == nil {
		panic("use routingutil.AllEdges instead of nil filter")
	}
	return &edgeIteratorImpl{
		EdgeIteratorStateImpl: *NewEdgeIteratorStateImpl(bg),
		filter:                filter,
	}
}

func (it *edgeIteratorImpl) SetBaseNode(baseNode int) util.EdgeIterator {
	nodePtr := it.store.ToNodePointer(baseNode)
	edgeRef := it.store.GetEdgeRef(nodePtr)
	it.nextEdgeID = edgeRef
	it.EdgeID = edgeRef
	it.BaseNode = baseNode
	return it
}

func (it *edgeIteratorImpl) Next() bool {
	for util.EdgeIsValid(it.nextEdgeID) {
		it.goToNext()
		if it.filter(it) {
			return true
		}
	}
	return false
}

func (it *edgeIteratorImpl) goToNext() {
	it.edgePointer = it.store.ToEdgePointer(it.nextEdgeID)
	it.EdgeID = it.nextEdgeID
	nodeA := it.store.GetNodeA(it.edgePointer)
	if it.BaseNode == nodeA {
		it.AdjNode = it.store.GetNodeB(it.edgePointer)
		it.Reverse = false
		it.nextEdgeID = it.store.GetLinkA(it.edgePointer)
	} else {
		it.AdjNode = nodeA
		it.Reverse = true
		it.nextEdgeID = it.store.GetLinkB(it.edgePointer)
	}
}

func (it *edgeIteratorImpl) Detach(reverseArg bool) util.EdgeIteratorState {
	if it.EdgeID == it.nextEdgeID {
		panic(fmt.Sprintf("call next before detaching (edgeId:%d vs. next %d)", it.EdgeID, it.nextEdgeID))
	}
	return it.EdgeIteratorStateImpl.Detach(reverseArg)
}

// allEdgeIterator iterates over all edges in the graph sequentially.
type allEdgeIterator struct {
	EdgeIteratorStateImpl
}

func newAllEdgeIterator(bg *BaseGraph) *allEdgeIterator {
	return &allEdgeIterator{
		EdgeIteratorStateImpl: *NewEdgeIteratorStateImpl(bg),
	}
}

func (it *allEdgeIterator) Length() int {
	return it.store.GetEdges()
}

func (it *allEdgeIterator) Next() bool {
	it.EdgeID++
	if it.EdgeID >= it.store.GetEdges() {
		return false
	}
	it.edgePointer = it.store.ToEdgePointer(it.EdgeID)
	it.BaseNode = it.store.GetNodeA(it.edgePointer)
	it.AdjNode = it.store.GetNodeB(it.edgePointer)
	it.Reverse = false
	return true
}

func (it *allEdgeIterator) Detach(reverseArg bool) util.EdgeIteratorState {
	if it.edgePointer < 0 {
		panic("call next before detaching")
	}
	iter := newAllEdgeIterator(it.baseGraph)
	iter.EdgeID = it.EdgeID
	iter.edgePointer = it.edgePointer
	if reverseArg {
		iter.Reverse = !it.Reverse
		iter.BaseNode = it.AdjNode
		iter.AdjNode = it.BaseNode
	} else {
		iter.Reverse = it.Reverse
		iter.BaseNode = it.BaseNode
		iter.AdjNode = it.AdjNode
	}
	return iter
}
