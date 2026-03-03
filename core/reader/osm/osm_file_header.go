package osm

import (
	"fmt"

	"gohopper/core/reader"
)

// OSMFileHeader represents the header of an OSM file.
type OSMFileHeader struct {
	reader.ReaderElement
}

func NewOSMFileHeader() *OSMFileHeader {
	return &OSMFileHeader{
		ReaderElement: reader.NewReaderElement(0, reader.TypeFileHeader),
	}
}

func (h *OSMFileHeader) String() string {
	return fmt.Sprintf("OSM File header:%v", h.GetTags())
}
