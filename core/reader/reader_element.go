package reader

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// ElementType identifies the kind of OSM element.
type ElementType int

const (
	TypeNode       ElementType = iota
	TypeWay
	TypeRelation
	TypeFileHeader
)

func (t ElementType) String() string {
	switch t {
	case TypeNode:
		return "NODE"
	case TypeWay:
		return "WAY"
	case TypeRelation:
		return "RELATION"
	case TypeFileHeader:
		return "FILEHEADER"
	default:
		return "UNKNOWN"
	}
}

// ReaderElement is the base for all OSM element types (node, way, relation, file header).
type ReaderElement struct {
	id         int64
	elemType   ElementType
	properties map[string]any
}

// NewReaderElement creates a ReaderElement with an empty tag map.
// Exported so that subpackages (e.g. reader/osm) can embed it.
func NewReaderElement(id int64, elemType ElementType) ReaderElement {
	if id < 0 {
		panic(fmt.Sprintf("Invalid OSM %s Id: %d; Ids must not be negative", elemType, id))
	}
	return ReaderElement{
		id:         id,
		elemType:   elemType,
		properties: make(map[string]any, 4),
	}
}

// NewReaderElementWithTags creates a ReaderElement using the given tag map directly.
func NewReaderElementWithTags(id int64, elemType ElementType, tags map[string]any) ReaderElement {
	if id < 0 {
		panic(fmt.Sprintf("Invalid OSM %s Id: %d; Ids must not be negative", elemType, id))
	}
	return ReaderElement{
		id:         id,
		elemType:   elemType,
		properties: tags,
	}
}

func (e *ReaderElement) GetID() int64        { return e.id }
func (e *ReaderElement) GetType() ElementType { return e.elemType }

// GetTags returns the underlying tag map directly.
func (e *ReaderElement) GetTags() map[string]any { return e.properties }

// SetTags replaces all tags. If tags is nil the existing tags are cleared.
func (e *ReaderElement) SetTags(tags map[string]any) {
	clear(e.properties)
	maps.Copy(e.properties, tags)
}

func (e *ReaderElement) HasTags() bool { return len(e.properties) > 0 }

// GetTag returns the value for key as a string, or "" if absent or not a string.
func (e *ReaderElement) GetTag(key string) string {
	v, ok := e.properties[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// GetTagWithDefault returns the value for key, or defaultValue if absent.
func (e *ReaderElement) GetTagWithDefault(key string, defaultValue any) any {
	v, ok := e.properties[key]
	if !ok {
		return defaultValue
	}
	return v
}

// SetTag sets a tag.
func (e *ReaderElement) SetTag(key string, value any) {
	e.properties[key] = value
}

// HasTag checks if the tag exists and matches any of the given values.
// If no values are provided, it checks for tag existence.
func (e *ReaderElement) HasTag(key string, values ...string) bool {
	if len(values) == 0 {
		_, ok := e.properties[key]
		return ok
	}
	v := e.GetTag(key)
	return slices.Contains(values, v)
}

// HasTagInCollection checks if the tag value is in the given collection.
func (e *ReaderElement) HasTagInCollection(key string, values map[string]struct{}) bool {
	v := e.GetTag(key)
	_, ok := values[v]
	return ok
}

// HasTagWithValue checks if the tag exists with the given value.
func (e *ReaderElement) HasTagWithValue(key string, value string) bool {
	return e.GetTag(key) == value
}

// HasTagFromKeys checks if any of the given keys have a value in the given values set.
func (e *ReaderElement) HasTagFromKeys(keys []string, values map[string]struct{}) bool {
	for _, k := range keys {
		v := e.GetTag(k)
		if _, ok := values[v]; ok {
			return true
		}
	}
	return false
}

// GetFirstValue returns the value of the first found key, or "" if none.
func (e *ReaderElement) GetFirstValue(keys []string) string {
	for _, k := range keys {
		v := e.GetTag(k)
		if v != "" {
			return v
		}
	}
	return ""
}

// GetFirstIndex returns the index of the first found key with a non-empty value, or -1.
func (e *ReaderElement) GetFirstIndex(keys []string) int {
	for i, k := range keys {
		v := e.GetTag(k)
		if v != "" {
			return i
		}
	}
	return -1
}

// RemoveTag removes a tag.
func (e *ReaderElement) RemoveTag(key string) {
	delete(e.properties, key)
}

// ClearTags removes all tags.
func (e *ReaderElement) ClearTags() {
	clear(e.properties)
}

func (e *ReaderElement) String() string {
	return tagsToString(e.properties)
}

func tagsToString(m map[string]any) string {
	if len(m) == 0 {
		return "<empty>"
	}
	var b strings.Builder
	for k, v := range m {
		fmt.Fprintf(&b, "%s=%v\n", k, v)
	}
	return b.String()
}
