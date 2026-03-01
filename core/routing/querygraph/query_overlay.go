package querygraph

import (
	"gohopper/core/util"
)

type EdgeChanges struct {
	AdditionalEdges []util.EdgeIteratorState
	RemovedEdges    []int
}

type QueryOverlay struct {
	virtualNodes            *util.PointList
	closestEdges            []int
	virtualEdges            []*VirtualEdgeIteratorState
	edgeChangesAtRealNodes  map[int]*EdgeChanges
}

func newQueryOverlay(numVirtualNodes int, is3D bool) *QueryOverlay {
	return &QueryOverlay{
		virtualNodes:           util.NewPointList(numVirtualNodes, is3D),
		virtualEdges:           make([]*VirtualEdgeIteratorState, 0, numVirtualNodes*2),
		closestEdges:           make([]int, 0, numVirtualNodes),
		edgeChangesAtRealNodes: make(map[int]*EdgeChanges, numVirtualNodes*3),
	}
}

func (qo *QueryOverlay) getNumVirtualEdges() int {
	return len(qo.virtualEdges)
}

func (qo *QueryOverlay) addVirtualEdge(ve *VirtualEdgeIteratorState) {
	qo.virtualEdges = append(qo.virtualEdges, ve)
}

func (qo *QueryOverlay) getVirtualEdge(edgeID int) *VirtualEdgeIteratorState {
	return qo.virtualEdges[edgeID]
}

func (qo *QueryOverlay) getVirtualEdges() []*VirtualEdgeIteratorState {
	return qo.virtualEdges
}

func (qo *QueryOverlay) getEdgeChangesAtRealNodes() map[int]*EdgeChanges {
	return qo.edgeChangesAtRealNodes
}

func (qo *QueryOverlay) getVirtualNodes() *util.PointList {
	return qo.virtualNodes
}

func (qo *QueryOverlay) getClosestEdges() []int {
	return qo.closestEdges
}
