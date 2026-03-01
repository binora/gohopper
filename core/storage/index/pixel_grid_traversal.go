package index

import (
	"math"

	"gohopper/core/util"
)

// PixelGridTraversal finds all grid cells intersected by a line segment using
// the "A Fast Voxel Traversal Algorithm for Ray Tracing" by John Amanatides
// and Andrew Woo (1987): http://www.cse.yorku.ca/~amana/research/grid.pdf
type PixelGridTraversal struct {
	parts  int
	bounds util.BBox
	deltaY float64
	deltaX float64
}

// NewPixelGridTraversal creates a new PixelGridTraversal for the given grid
// resolution and bounding box.
func NewPixelGridTraversal(parts int, bounds util.BBox) *PixelGridTraversal {
	return &PixelGridTraversal{
		parts:  parts,
		bounds: bounds,
		deltaY: (bounds.MaxLat - bounds.MinLat) / float64(parts),
		deltaX: (bounds.MaxLon - bounds.MinLon) / float64(parts),
	}
}

// Traverse calls consumer for every grid cell that the line segment from a to
// b passes through. The coordinates use [2]float64 where index 0 is x (lon)
// and index 1 is y (lat), matching the JTS Coordinate convention.
func (p *PixelGridTraversal) Traverse(a, b [2]float64, consumer func(x, y int)) {
	ax := a[0] - p.bounds.MinLon
	ay := a[1] - p.bounds.MinLat
	bx := b[0] - p.bounds.MinLon
	by := b[1] - p.bounds.MinLat

	stepX := 1
	if ax >= bx {
		stepX = -1
	}
	stepY := 1
	if ay >= by {
		stepY = -1
	}
	tDeltaX := p.deltaX / math.Abs(bx-ax)
	tDeltaY := p.deltaY / math.Abs(by-ay)

	// Bounding with parts-1 only concerns the case where we are exactly on
	// the bounding box edge.
	x := min(int(ax/p.deltaX), p.parts-1)
	y := min(int(ay/p.deltaY), p.parts-1)
	x2 := min(int(bx/p.deltaX), p.parts-1)
	y2 := min(int(by/p.deltaY), p.parts-1)

	stepXOffset := 1
	if stepX < 0 {
		stepXOffset = 0
	}
	stepYOffset := 1
	if stepY < 0 {
		stepYOffset = 0
	}
	tMaxX := (float64(x+stepXOffset)*p.deltaX - ax) / (bx - ax)
	tMaxY := (float64(y+stepYOffset)*p.deltaY - ay) / (by - ay)

	consumer(x, y)
	for y != y2 || x != x2 {
		if (tMaxX < tMaxY || y == y2) && x != x2 {
			tMaxX += tDeltaX
			x += stepX
		} else {
			tMaxY += tDeltaY
			y += stepY
		}
		consumer(x, y)
	}
}
