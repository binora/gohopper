package osm

// SegmentNode represents a node within a way segment during import processing.
type SegmentNode struct {
	OSMNodeID int64
	ID        int64
	Tags      map[string]any
}
