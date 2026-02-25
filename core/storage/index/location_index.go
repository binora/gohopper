package index

import "gohopper/core/util"

type Snap struct {
	Valid        bool
	SnappedPoint util.GHPoint
}

func (s Snap) IsValid() bool {
	return s.Valid
}

type LocationIndex struct{}

func NewLocationIndex() *LocationIndex {
	return &LocationIndex{}
}

func (l *LocationIndex) FindClosest(lat, lon float64) Snap {
	// Placeholder implementation: exact point pass-through until real graph index is implemented.
	return Snap{Valid: true, SnappedPoint: util.GHPoint{Lat: lat, Lon: lon}}
}
