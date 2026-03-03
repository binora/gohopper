package parsers

import (
	"gohopper/core/reader"
	"gohopper/core/storage"
)

// RelationTagParser extends TagParser with the ability to process relation tags.
// Relation flags computed by HandleRelationTags are passed to HandleWayTags per edge.
type RelationTagParser interface {
	TagParser
	HandleRelationTags(relFlags *storage.IntsRef, relation *reader.ReaderRelation)
}
