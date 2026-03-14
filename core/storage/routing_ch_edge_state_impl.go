package storage

import (
	"fmt"
	"math"

	"gohopper/core/util"
)

type routingCHEdgeIteratorStateImpl struct {
	store           *CHStorage
	baseGraph       *BaseGraph
	weighting       CHWeighting
	edgeID          int
	baseNode        int
	adjNode         int
	baseEdgeState   *EdgeIteratorStateImpl
	shortcutPointer int64
}

func newRoutingCHEdgeIteratorStateImpl(store *CHStorage, bg *BaseGraph, baseEdgeState *EdgeIteratorStateImpl, w CHWeighting) *routingCHEdgeIteratorStateImpl {
	return &routingCHEdgeIteratorStateImpl{
		store:           store,
		baseGraph:       bg,
		baseEdgeState:   baseEdgeState,
		weighting:       w,
		edgeID:          -1,
		shortcutPointer: -1,
	}
}

func (s *routingCHEdgeIteratorStateImpl) init(edge, expectedAdjNode int) bool {
	if edge < 0 || edge >= s.baseGraph.GetEdges()+s.store.GetShortcuts() {
		panic(fmt.Sprintf("edge must be in bounds: [0,%d[", s.baseGraph.GetEdges()+s.store.GetShortcuts()))
	}
	s.edgeID = edge
	if s.IsShortcut() {
		s.shortcutPointer = s.store.ToShortcutPointer(edge - s.baseGraph.GetEdges())
		s.baseNode = s.store.GetNodeA(s.shortcutPointer)
		s.adjNode = s.store.GetNodeB(s.shortcutPointer)

		if expectedAdjNode == s.adjNode || expectedAdjNode == math.MinInt32 {
			return true
		} else if expectedAdjNode == s.baseNode {
			s.baseNode = s.adjNode
			s.adjNode = expectedAdjNode
			return true
		}
		return false
	}
	return s.baseEdgeState.Init(edge, expectedAdjNode)
}

func (s *routingCHEdgeIteratorStateImpl) GetEdge() int {
	return s.edgeID
}

func (s *routingCHEdgeIteratorStateImpl) GetOrigEdge() int {
	if s.IsShortcut() {
		return util.NoEdge
	}
	return s.edgeState().GetEdge()
}

func (s *routingCHEdgeIteratorStateImpl) GetOrigEdgeKeyFirst() int {
	if !s.IsShortcut() || !s.store.IsEdgeBased() {
		return s.edgeState().GetEdgeKey()
	}
	return s.store.GetOrigEdgeKeyFirst(s.shortcutPointer)
}

func (s *routingCHEdgeIteratorStateImpl) GetOrigEdgeKeyLast() int {
	if !s.IsShortcut() || !s.store.IsEdgeBased() {
		return s.edgeState().GetEdgeKey()
	}
	return s.store.GetOrigEdgeKeyLast(s.shortcutPointer)
}

func (s *routingCHEdgeIteratorStateImpl) GetBaseNode() int {
	if s.IsShortcut() {
		return s.baseNode
	}
	return s.edgeState().GetBaseNode()
}

func (s *routingCHEdgeIteratorStateImpl) GetAdjNode() int {
	if s.IsShortcut() {
		return s.adjNode
	}
	return s.edgeState().GetAdjNode()
}

func (s *routingCHEdgeIteratorStateImpl) IsShortcut() bool {
	return s.edgeID >= s.baseGraph.GetEdges()
}

func (s *routingCHEdgeIteratorStateImpl) GetSkippedEdge1() int {
	s.checkShortcut(true, "GetSkippedEdge1")
	return s.store.GetSkippedEdge1(s.shortcutPointer)
}

func (s *routingCHEdgeIteratorStateImpl) GetSkippedEdge2() int {
	s.checkShortcut(true, "GetSkippedEdge2")
	return s.store.GetSkippedEdge2(s.shortcutPointer)
}

func (s *routingCHEdgeIteratorStateImpl) GetWeight(reverse bool) float64 {
	if s.IsShortcut() {
		return s.store.GetWeight(s.shortcutPointer)
	}
	return s.getOrigEdgeWeight(reverse)
}

func (s *routingCHEdgeIteratorStateImpl) getOrigEdgeWeight(reverse bool) float64 {
	return s.weighting.CalcEdgeWeight(s.getBaseGraphEdgeState(), reverse)
}

func (s *routingCHEdgeIteratorStateImpl) getBaseGraphEdgeState() util.EdgeIteratorState {
	s.checkShortcut(false, "getBaseGraphEdgeState")
	return s.edgeState()
}

func (s *routingCHEdgeIteratorStateImpl) edgeState() util.EdgeIteratorState {
	return s.baseEdgeState
}

func (s *routingCHEdgeIteratorStateImpl) checkShortcut(shouldBeShortcut bool, methodName string) {
	if s.IsShortcut() {
		if !shouldBeShortcut {
			panic(fmt.Sprintf("Cannot call %s on shortcut %d", methodName, s.GetEdge()))
		}
	} else if shouldBeShortcut {
		panic(fmt.Sprintf("Method %s only for shortcuts %d", methodName, s.GetEdge()))
	}
}
