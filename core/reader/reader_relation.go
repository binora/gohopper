package reader

import "fmt"

// MemberType identifies the kind of relation member.
type MemberType = ElementType

// Member is a relation member referencing another OSM element.
type Member struct {
	Type MemberType
	Ref  int64
	Role string
}

func (m Member) String() string {
	return fmt.Sprintf("Member %s:%d", m.Type, m.Ref)
}

// ReaderRelation represents an OSM relation.
type ReaderRelation struct {
	ReaderElement
	Members []Member
}

func NewReaderRelation(id int64) *ReaderRelation {
	return &ReaderRelation{
		ReaderElement: NewReaderElementWithTags(id, TypeRelation, make(map[string]any, 2)),
	}
}

func (r *ReaderRelation) GetMembers() []Member { return r.Members }

// IsMetaRelation returns true if any member is itself a relation.
func (r *ReaderRelation) IsMetaRelation() bool {
	for _, m := range r.Members {
		if m.Type == TypeRelation {
			return true
		}
	}
	return false
}

// Add adds a member to the relation.
func (r *ReaderRelation) Add(m Member) {
	r.Members = append(r.Members, m)
}

func (r *ReaderRelation) String() string {
	return fmt.Sprintf("Relation (%d, %d members)", r.id, len(r.Members))
}
