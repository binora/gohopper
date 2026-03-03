package osm

import "gohopper/core/reader"

// ElementHandler processes OSM elements during file reading.
type ElementHandler interface {
	HandleNode(node *reader.ReaderNode)
	HandleWay(way *reader.ReaderWay)
	HandleRelation(relation *reader.ReaderRelation)
	HandleFileHeader(header *OSMFileHeader)
	OnFinish()
}
