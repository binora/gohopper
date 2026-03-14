package storage

import "fmt"

type RoutingCHGraph interface {
	GetNodes() int
	GetEdges() int
	GetShortcuts() int
	CreateInEdgeExplorer() RoutingCHEdgeExplorer
	CreateOutEdgeExplorer() RoutingCHEdgeExplorer
	GetEdgeIteratorState(chEdge, adjNode int) RoutingCHEdgeIteratorState
	GetLevel(node int) int
	GetTurnWeight(inEdge, viaNode, outEdge int) float64
	GetBaseGraph() Graph
	HasTurnCosts() bool
	IsEdgeBased() bool
	GetWeighting() CHWeighting
	Close()
}

type RoutingCHGraphImpl struct {
	baseGraph *BaseGraph
	chStorage *CHStorage
	weighting CHWeighting
}

func NewRoutingCHGraph(bg *BaseGraph, chStorage *CHStorage, w CHWeighting) *RoutingCHGraphImpl {
	if w.HasTurnCosts() && !chStorage.IsEdgeBased() {
		panic("Weighting has turn costs, but CHStorage is node-based")
	}
	return &RoutingCHGraphImpl{
		baseGraph: bg,
		chStorage: chStorage,
		weighting: w,
	}
}

func NewRoutingCHGraphFromConfig(bg *BaseGraph, chStorage *CHStorage, w CHWeighting) *RoutingCHGraphImpl {
	return NewRoutingCHGraph(bg, chStorage, w)
}

func (g *RoutingCHGraphImpl) GetNodes() int {
	return g.baseGraph.GetNodes()
}

func (g *RoutingCHGraphImpl) GetEdges() int {
	return g.baseGraph.GetEdges() + g.chStorage.GetShortcuts()
}

func (g *RoutingCHGraphImpl) GetShortcuts() int {
	return g.chStorage.GetShortcuts()
}

func (g *RoutingCHGraphImpl) CreateInEdgeExplorer() RoutingCHEdgeExplorer {
	return newRoutingCHEdgeIteratorIn(g.chStorage, g.baseGraph, g.weighting)
}

func (g *RoutingCHGraphImpl) CreateOutEdgeExplorer() RoutingCHEdgeExplorer {
	return newRoutingCHEdgeIteratorOut(g.chStorage, g.baseGraph, g.weighting)
}

func (g *RoutingCHGraphImpl) GetEdgeIteratorState(chEdge, adjNode int) RoutingCHEdgeIteratorState {
	edgeState := newRoutingCHEdgeIteratorStateImpl(g.chStorage, g.baseGraph, NewEdgeIteratorStateImpl(g.baseGraph), g.weighting)
	if edgeState.init(chEdge, adjNode) {
		return edgeState
	}
	return nil
}

func (g *RoutingCHGraphImpl) GetLevel(node int) int {
	return g.chStorage.GetLevel(g.chStorage.ToNodePointer(node))
}

func (g *RoutingCHGraphImpl) GetBaseGraph() Graph {
	return g.baseGraph
}

func (g *RoutingCHGraphImpl) GetWeighting() CHWeighting {
	return g.weighting
}

func (g *RoutingCHGraphImpl) HasTurnCosts() bool {
	return g.weighting.HasTurnCosts()
}

func (g *RoutingCHGraphImpl) IsEdgeBased() bool {
	return g.chStorage.IsEdgeBased()
}

func (g *RoutingCHGraphImpl) GetTurnWeight(edgeFrom, nodeVia, edgeTo int) float64 {
	return g.weighting.CalcTurnWeight(edgeFrom, nodeVia, edgeTo)
}

func (g *RoutingCHGraphImpl) Close() {
	if !g.baseGraph.Store.IsClosed() {
		g.baseGraph.Close()
	}
	g.chStorage.Close()
}

func (g *RoutingCHGraphImpl) GetCHStorage() *CHStorage {
	return g.chStorage
}

func (g *RoutingCHGraphImpl) String() string {
	return fmt.Sprintf("RoutingCHGraph{nodes=%d, edges=%d, shortcuts=%d}", g.GetNodes(), g.GetEdges(), g.GetShortcuts())
}
