package querygraph

import (
	"fmt"

	"gohopper/core/util"
)

const (
	BaseSnap = 0
	SnapBase = 1
	SnapAdj  = 2
	AdjSnap  = 3
)

func buildEdgeChanges(closestEdges []int, virtualEdges []*VirtualEdgeIteratorState, firstVirtualNodeID int, edgeChangesAtRealNodes map[int]*EdgeChanges) {
	if len(edgeChangesAtRealNodes) != 0 {
		panic("real node modifications need to be empty")
	}
	b := &edgeChangeBuilder{
		closestEdges:           closestEdges,
		virtualEdges:           virtualEdges,
		firstVirtualNodeID:     firstVirtualNodeID,
		edgeChangesAtRealNodes: edgeChangesAtRealNodes,
	}
	b.build()
}

type edgeChangeBuilder struct {
	closestEdges           []int
	virtualEdges           []*VirtualEdgeIteratorState
	edgeChangesAtRealNodes map[int]*EdgeChanges
	firstVirtualNodeID     int
}

func (b *edgeChangeBuilder) build() {
	towerNodesToChange := make(map[int]bool, b.numVirtualNodes())

	for i := range b.numVirtualNodes() {
		baseRevEdge := b.getVirtualEdge(i*4 + SnapBase)
		towerNode := baseRevEdge.GetAdjNode()
		if !b.isVirtualNode(towerNode) {
			towerNodesToChange[towerNode] = true
			b.addVirtualEdges(true, towerNode, i)
		}

		adjEdge := b.getVirtualEdge(i*4 + SnapAdj)
		towerNode = adjEdge.GetAdjNode()
		if !b.isVirtualNode(towerNode) {
			towerNodesToChange[towerNode] = true
			b.addVirtualEdges(false, towerNode, i)
		}
	}

	for node := range towerNodesToChange {
		b.addRemovedEdges(node)
	}
}

func (b *edgeChangeBuilder) addVirtualEdges(base bool, node, virtNode int) {
	edgeChanges, ok := b.edgeChangesAtRealNodes[node]
	if !ok {
		edgeChanges = &EdgeChanges{
			AdditionalEdges: make([]util.EdgeIteratorState, 0, 2),
			RemovedEdges:    make([]int, 0, 2),
		}
		b.edgeChangesAtRealNodes[node] = edgeChanges
	}
	offset := AdjSnap
	if base {
		offset = BaseSnap
	}
	edgeChanges.AdditionalEdges = append(edgeChanges.AdditionalEdges, b.getVirtualEdge(virtNode*4+offset))
}

func (b *edgeChangeBuilder) addRemovedEdges(towerNode int) {
	if b.isVirtualNode(towerNode) {
		panic(fmt.Sprintf("node should not be virtual: %d", towerNode))
	}
	edgeChanges := b.edgeChangesAtRealNodes[towerNode]
	for _, existingEdge := range edgeChanges.AdditionalEdges {
		edgeChanges.RemovedEdges = append(edgeChanges.RemovedEdges, b.getClosestEdge(existingEdge.GetAdjNode()))
	}
}

func (b *edgeChangeBuilder) isVirtualNode(nodeID int) bool {
	return nodeID >= b.firstVirtualNodeID
}

func (b *edgeChangeBuilder) numVirtualNodes() int {
	return len(b.closestEdges)
}

func (b *edgeChangeBuilder) getClosestEdge(node int) int {
	return b.closestEdges[node-b.firstVirtualNodeID]
}

func (b *edgeChangeBuilder) getVirtualEdge(virtualEdgeID int) *VirtualEdgeIteratorState {
	return b.virtualEdges[virtualEdgeID]
}
