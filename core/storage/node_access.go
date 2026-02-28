package storage

// NodeAccess provides access to node properties (lat/lon/elevation/turn costs).
type NodeAccess interface {
	SetNode(nodeId int, lat, lon, ele float64)
	GetLat(nodeId int) float64
	GetLon(nodeId int) float64
	GetEle(nodeId int) float64
	Is3D() bool
	Dimension() int
	EnsureNode(nodeId int)
	GetTurnCostIndex(nodeId int) int
	SetTurnCostIndex(nodeId int, value int)
}

// ghNodeAccess adapts BaseGraphNodesAndEdges to the NodeAccess interface.
type ghNodeAccess struct {
	store *BaseGraphNodesAndEdges
}

func newGHNodeAccess(store *BaseGraphNodesAndEdges) *ghNodeAccess {
	return &ghNodeAccess{store: store}
}

func (na *ghNodeAccess) EnsureNode(nodeId int) {
	na.store.EnsureNodeCapacity(nodeId)
}

func (na *ghNodeAccess) SetNode(nodeId int, lat, lon, ele float64) {
	na.store.EnsureNodeCapacity(nodeId)
	ptr := na.store.ToNodePointer(nodeId)
	na.store.SetLat(ptr, lat)
	na.store.SetLon(ptr, lon)
	if na.store.WithElevation() {
		na.store.SetEle(ptr, ele)
		na.store.Bounds.Update3D(lat, lon, ele)
	} else {
		na.store.Bounds.Update(lat, lon)
	}
}

func (na *ghNodeAccess) GetLat(nodeId int) float64 {
	return na.store.GetLat(na.store.ToNodePointer(nodeId))
}

func (na *ghNodeAccess) GetLon(nodeId int) float64 {
	return na.store.GetLon(na.store.ToNodePointer(nodeId))
}

func (na *ghNodeAccess) GetEle(nodeId int) float64 {
	if !na.store.WithElevation() {
		panic("elevation is disabled")
	}
	return na.store.GetEle(na.store.ToNodePointer(nodeId))
}

func (na *ghNodeAccess) SetTurnCostIndex(nodeId int, turnCostIndex int) {
	if !na.store.WithTurnCosts() {
		panic("this graph does not support turn costs")
	}
	na.store.EnsureNodeCapacity(nodeId)
	na.store.SetTurnCostRef(na.store.ToNodePointer(nodeId), turnCostIndex)
}

func (na *ghNodeAccess) GetTurnCostIndex(nodeId int) int {
	if !na.store.WithTurnCosts() {
		panic("this graph does not support turn costs")
	}
	return na.store.GetTurnCostRef(na.store.ToNodePointer(nodeId))
}

func (na *ghNodeAccess) Is3D() bool {
	return na.store.WithElevation()
}

func (na *ghNodeAccess) Dimension() int {
	if na.store.WithElevation() {
		return 3
	}
	return 2
}
