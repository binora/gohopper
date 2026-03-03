package parsers

import (
	"gohopper/core/reader"
	"gohopper/core/routing/ev"
	"gohopper/core/storage"
)

// TagParser is applied to each edge during import to populate encoded values from OSM tags.
type TagParser interface {
	HandleWayTags(edgeID int, edgeIntAccess ev.EdgeIntAccess, way *reader.ReaderWay, relationFlags *storage.IntsRef)
}
