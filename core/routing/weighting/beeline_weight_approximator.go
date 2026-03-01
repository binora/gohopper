package weighting

import (
	"gohopper/core/storage"
	"gohopper/core/util"
)

var _ WeightApproximator = (*BeelineWeightApproximator)(nil)

// BeelineWeightApproximator estimates the remaining weight to a goal node
// using the beeline (straight-line) distance scaled by the minimum weight per distance.
type BeelineWeightApproximator struct {
	nodeAccess           storage.NodeAccess
	weighting            Weighting
	minWeightPerDistance float64
	distCalc             util.DistanceCalc
	toLat, toLon         float64
	epsilon              float64
}

func NewBeelineWeightApproximator(nodeAccess storage.NodeAccess, weighting Weighting) *BeelineWeightApproximator {
	return &BeelineWeightApproximator{
		nodeAccess:           nodeAccess,
		weighting:            weighting,
		minWeightPerDistance: weighting.CalcMinWeightPerDistance(),
		distCalc:             util.DistEarth,
		epsilon:              1,
	}
}

func (b *BeelineWeightApproximator) SetTo(toNode int) {
	b.toLat = b.nodeAccess.GetLat(toNode)
	b.toLon = b.nodeAccess.GetLon(toNode)
}

func (b *BeelineWeightApproximator) SetEpsilon(epsilon float64) *BeelineWeightApproximator {
	b.epsilon = epsilon
	return b
}

func (b *BeelineWeightApproximator) Reverse() WeightApproximator {
	return NewBeelineWeightApproximator(b.nodeAccess, b.weighting).
		SetDistanceCalc(b.distCalc).
		SetEpsilon(b.epsilon)
}

func (b *BeelineWeightApproximator) GetSlack() float64 {
	return 0
}

func (b *BeelineWeightApproximator) Approximate(fromNode int) float64 {
	fromLat := b.nodeAccess.GetLat(fromNode)
	fromLon := b.nodeAccess.GetLon(fromNode)
	dist := b.distCalc.CalcDist(b.toLat, b.toLon, fromLat, fromLon)
	return dist * b.minWeightPerDistance * b.epsilon
}

func (b *BeelineWeightApproximator) SetDistanceCalc(dc util.DistanceCalc) *BeelineWeightApproximator {
	b.distCalc = dc
	return b
}

func (b *BeelineWeightApproximator) String() string {
	return "beeline"
}
