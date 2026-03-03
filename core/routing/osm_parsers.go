package routing

import (
	"slices"

	"gohopper/core/reader"
	"gohopper/core/routing/ev"
	"gohopper/core/routing/parsers"
	"gohopper/core/storage"
)

// OSMParsers holds the tag parsers used during OSM import to populate encoded values.
type OSMParsers struct {
	ignoredHighways    []string
	wayTagParsers      []parsers.TagParser
	relationTagParsers []parsers.RelationTagParser
	relConfig          *ev.InitializerConfig
}

func NewOSMParsers() *OSMParsers {
	return &OSMParsers{
		relConfig: ev.NewInitializerConfig(),
	}
}

func (p *OSMParsers) AddIgnoredHighway(highway string) *OSMParsers {
	p.ignoredHighways = append(p.ignoredHighways, highway)
	return p
}

func (p *OSMParsers) AddWayTagParser(tp parsers.TagParser) *OSMParsers {
	p.wayTagParsers = append(p.wayTagParsers, tp)
	return p
}

func (p *OSMParsers) AddRelationTagParser(tp parsers.RelationTagParser) *OSMParsers {
	p.relationTagParsers = append(p.relationTagParsers, tp)
	return p
}

func (p *OSMParsers) GetWayTagParsers() []parsers.TagParser              { return p.wayTagParsers }
func (p *OSMParsers) GetRelationTagParsers() []parsers.RelationTagParser { return p.relationTagParsers }
func (p *OSMParsers) GetIgnoredHighways() []string                       { return p.ignoredHighways }

// AcceptWay returns true if this way should be imported.
func (p *OSMParsers) AcceptWay(way *reader.ReaderWay) bool {
	highway := way.GetTag("highway")
	if highway != "" {
		return !slices.Contains(p.ignoredHighways, highway)
	}
	if way.GetTag("route") != "" {
		return true
	}
	if way.GetTag("man_made") == "pier" {
		return true
	}
	if way.GetTag("railway") == "platform" {
		return true
	}
	return false
}

// HandleRelationTags dispatches to all relation tag parsers.
func (p *OSMParsers) HandleRelationTags(relation *reader.ReaderRelation, relFlags *storage.IntsRef) {
	for _, rtp := range p.relationTagParsers {
		rtp.HandleRelationTags(relFlags, relation)
	}
}

// HandleWayTags dispatches to all relation tag parsers (for way tags), then all way tag parsers.
func (p *OSMParsers) HandleWayTags(edgeID int, edgeIntAccess ev.EdgeIntAccess, way *reader.ReaderWay, relationFlags *storage.IntsRef) {
	for _, rtp := range p.relationTagParsers {
		rtp.HandleWayTags(edgeID, edgeIntAccess, way, relationFlags)
	}
	for _, tp := range p.wayTagParsers {
		tp.HandleWayTags(edgeID, edgeIntAccess, way, relationFlags)
	}
}

// CreateRelationFlags creates a new IntsRef for relation flag storage.
func (p *OSMParsers) CreateRelationFlags() *storage.IntsRef {
	requiredInts := p.relConfig.GetRequiredInts()
	if requiredInts > 2 {
		panic("relation flags require more than 2 ints, which is not supported")
	}
	return storage.NewIntsRef(2)
}
