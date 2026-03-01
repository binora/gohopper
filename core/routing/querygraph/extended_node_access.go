package querygraph

import (
	"gohopper/core/storage"
	"gohopper/core/util"
)

var _ storage.NodeAccess = (*ExtendedNodeAccess)(nil)

type ExtendedNodeAccess struct {
	base                  storage.NodeAccess
	virtualNodes          *util.PointList
	firstAdditionalNodeID int
}

func NewExtendedNodeAccess(base storage.NodeAccess, virtualNodes *util.PointList, firstAdditionalNodeID int) *ExtendedNodeAccess {
	return &ExtendedNodeAccess{
		base:                  base,
		virtualNodes:          virtualNodes,
		firstAdditionalNodeID: firstAdditionalNodeID,
	}
}

func (e *ExtendedNodeAccess) SetNode(nodeID int, lat, lon, ele float64) {
	panic("not supported for ExtendedNodeAccess")
}

func (e *ExtendedNodeAccess) GetLat(nodeID int) float64 {
	if nodeID < e.firstAdditionalNodeID {
		return e.base.GetLat(nodeID)
	}
	return e.virtualNodes.GetLat(nodeID - e.firstAdditionalNodeID)
}

func (e *ExtendedNodeAccess) GetLon(nodeID int) float64 {
	if nodeID < e.firstAdditionalNodeID {
		return e.base.GetLon(nodeID)
	}
	return e.virtualNodes.GetLon(nodeID - e.firstAdditionalNodeID)
}

func (e *ExtendedNodeAccess) GetEle(nodeID int) float64 {
	if nodeID < e.firstAdditionalNodeID {
		return e.base.GetEle(nodeID)
	}
	return e.virtualNodes.GetEle(nodeID - e.firstAdditionalNodeID)
}

func (e *ExtendedNodeAccess) Is3D() bool {
	return e.base.Is3D()
}

func (e *ExtendedNodeAccess) Dimension() int {
	return e.base.Dimension()
}

func (e *ExtendedNodeAccess) EnsureNode(_ int) {
	panic("not supported for ExtendedNodeAccess")
}

func (e *ExtendedNodeAccess) GetTurnCostIndex(nodeID int) int {
	if nodeID < e.firstAdditionalNodeID {
		return e.base.GetTurnCostIndex(nodeID)
	}
	return 0
}

func (e *ExtendedNodeAccess) SetTurnCostIndex(_ int, _ int) {
	panic("not supported for ExtendedNodeAccess")
}
