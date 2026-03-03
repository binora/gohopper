package reader

import "fmt"

// ReaderNode represents an OSM node with coordinates.
type ReaderNode struct {
	ReaderElement
	Lat float64
	Lon float64
}

func NewReaderNode(id int64, lat, lon float64) *ReaderNode {
	return &ReaderNode{
		ReaderElement: NewReaderElement(id, TypeNode),
		Lat:           lat,
		Lon:           lon,
	}
}

func NewReaderNodeWithTags(id int64, lat, lon float64, tags map[string]any) *ReaderNode {
	return &ReaderNode{
		ReaderElement: NewReaderElementWithTags(id, TypeNode, tags),
		Lat:           lat,
		Lon:           lon,
	}
}

func (n *ReaderNode) String() string {
	s := fmt.Sprintf("Node: %d lat=%f lon=%f", n.id, n.Lat, n.Lon)
	if n.HasTags() {
		s += "\n" + tagsToString(n.properties)
	}
	return s
}
