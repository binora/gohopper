package storage

// NodeAccess provides access to node properties (lat/lon/elevation/turn costs).
type NodeAccess interface {
	SetNode(nodeID int, lat, lon, ele float64)
	GetLat(nodeID int) float64
	GetLon(nodeID int) float64
	GetEle(nodeID int) float64
	Is3D() bool
	Dimension() int
	EnsureNode(nodeID int)
	GetTurnCostIndex(nodeID int) int
	SetTurnCostIndex(nodeID int, value int)
}

// ghNodeAccess adapts BaseGraphNodesAndEdges to the NodeAccess interface.
type ghNodeAccess struct {
	store *BaseGraphNodesAndEdges
}

func newGHNodeAccess(store *BaseGraphNodesAndEdges) *ghNodeAccess {
	return &ghNodeAccess{store: store}
}

func (n *ghNodeAccess) EnsureNode(nodeID int) {
	n.store.EnsureNodeCapacity(nodeID)
}

func (n *ghNodeAccess) SetNode(nodeID int, lat, lon, ele float64) {
	n.store.EnsureNodeCapacity(nodeID)
	ptr := n.store.ToNodePointer(nodeID)
	n.store.SetLat(ptr, lat)
	n.store.SetLon(ptr, lon)
	if n.store.WithElevation() {
		n.store.SetEle(ptr, ele)
		n.store.Bounds.Update3D(lat, lon, ele)
	} else {
		n.store.Bounds.Update(lat, lon)
	}
}

func (n *ghNodeAccess) GetLat(nodeID int) float64 {
	return n.store.GetLat(n.store.ToNodePointer(nodeID))
}

func (n *ghNodeAccess) GetLon(nodeID int) float64 {
	return n.store.GetLon(n.store.ToNodePointer(nodeID))
}

func (n *ghNodeAccess) GetEle(nodeID int) float64 {
	if !n.store.WithElevation() {
		panic("elevation is disabled")
	}
	return n.store.GetEle(n.store.ToNodePointer(nodeID))
}

func (n *ghNodeAccess) SetTurnCostIndex(nodeID int, tcIndex int) {
	if !n.store.WithTurnCosts() {
		panic("this graph does not support turn costs")
	}
	n.store.EnsureNodeCapacity(nodeID)
	n.store.SetTurnCostRef(n.store.ToNodePointer(nodeID), tcIndex)
}

func (n *ghNodeAccess) GetTurnCostIndex(nodeID int) int {
	if !n.store.WithTurnCosts() {
		panic("this graph does not support turn costs")
	}
	return n.store.GetTurnCostRef(n.store.ToNodePointer(nodeID))
}

func (n *ghNodeAccess) Is3D() bool {
	return n.store.WithElevation()
}

func (n *ghNodeAccess) Dimension() int {
	if n.store.WithElevation() {
		return 3
	}
	return 2
}
