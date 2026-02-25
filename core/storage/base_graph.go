package storage

import "gohopper/core/util"

type BBox struct {
	MinLon float64
	MinLat float64
	MaxLon float64
	MaxLat float64
}

func (b BBox) Contains(lat, lon float64) bool {
	return lat >= b.MinLat && lat <= b.MaxLat && lon >= b.MinLon && lon <= b.MaxLon
}

type BaseGraph struct {
	bounds BBox
}

func NewBaseGraph() *BaseGraph {
	// world bounds default until real import populates exact region bounds
	return &BaseGraph{bounds: BBox{MinLon: -180, MinLat: -90, MaxLon: 180, MaxLat: 90}}
}

func (g *BaseGraph) SetBounds(points []util.GHPoint) {
	bbox := util.CalcBBox(points)
	g.bounds = BBox{MinLon: bbox[0], MinLat: bbox[1], MaxLon: bbox[2], MaxLat: bbox[3]}
}

func (g *BaseGraph) GetBounds() BBox {
	return g.bounds
}
