package storage

import "gohopper/core/util"

type BaseGraph struct {
	bounds util.BBox
}

func NewBaseGraph() *BaseGraph {
	// world bounds default until real import populates exact region bounds
	return &BaseGraph{bounds: util.NewBBox(-180, 180, -90, 90)}
}

func (g *BaseGraph) SetBounds(points []util.GHPoint) {
	g.bounds = util.CalcBBox(points)
}

func (g *BaseGraph) GetBounds() util.BBox {
	return g.bounds
}
