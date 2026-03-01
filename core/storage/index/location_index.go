package index

import (
	routeutil "gohopper/core/routing/util"
	"gohopper/core/util"
)

// LocationIndex provides a way to map real-world coordinates (lat, lon) to
// internal IDs/indices of a memory-efficient graph. Implementations of
// FindClosest must be thread-safe.
type LocationIndex interface {
	// FindClosest returns the closest Snap for the specified location. The
	// edgeFilter controls which edges are valid candidates (e.g. filtering
	// away car-only results for a bike search). If nothing is found the
	// returned Snap's IsValid method will return false.
	FindClosest(lat, lon float64, edgeFilter routeutil.EdgeFilter) *Snap

	// Query explores the LocationIndex with the specified TileFilter and
	// Visitor. It visits only stored edges (each at most once), limited by
	// the tile filter. A few edges slightly outside the query area may also
	// be returned; callers can perform an explicit BBox check to exclude them.
	Query(tileFilter TileFilter, visitor Visitor)

	// Close releases resources held by the index.
	Close()
}

// TileFilter controls which tiles are accepted during a LocationIndex query.
type TileFilter interface {
	// AcceptAll returns true if all edges within the given bounding box shall
	// be accepted.
	AcceptAll(tile util.BBox) bool

	// AcceptPartially returns true if edges within the given bounding box
	// shall potentially be accepted. In this case the tile filter will be
	// applied again for smaller bounding boxes on a lower level. If this is
	// the lowest level already, all edges will be accepted.
	AcceptPartially(tile util.BBox) bool
}

// Visitor allows visiting edges stored in the LocationIndex.
type Visitor interface {
	// OnEdge is called for each visited edge.
	OnEdge(edgeID int)

	// IsTileInfo returns true if OnTile should be called.
	IsTileInfo() bool

	// OnTile is called (when IsTileInfo returns true) with the bounding box
	// and depth of each visited tile.
	OnTile(bbox util.BBox, depth int)
}

// QueryBBox is a convenience function that queries the LocationIndex using a
// BBox-based TileFilter. This mirrors the default query(BBox, Visitor) method
// in the Java interface, which Go interfaces cannot express directly.
func QueryBBox(idx LocationIndex, queryBBox util.BBox, visitor Visitor) {
	idx.Query(CreateBBoxTileFilter(queryBBox), visitor)
}

// CreateBBoxTileFilter creates a TileFilter that accepts tiles fully contained
// in or intersecting the given bounding box.
func CreateBBoxTileFilter(bbox util.BBox) TileFilter {
	return &bboxTileFilter{bbox: bbox}
}

type bboxTileFilter struct {
	bbox util.BBox
}

func (f *bboxTileFilter) AcceptAll(tile util.BBox) bool {
	return f.bbox.ContainsBBox(tile)
}

func (f *bboxTileFilter) AcceptPartially(tile util.BBox) bool {
	return f.bbox.Intersects(tile)
}

