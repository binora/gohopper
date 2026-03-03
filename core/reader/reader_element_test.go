package reader

import "testing"

func TestHasTag(t *testing.T) {
	w := NewReaderWay(1)
	w.SetTag("surface", "something")
	if !w.HasTag("surface", "now", "something") {
		t.Fatal("expected tag match")
	}
	if w.HasTag("surface", "now", "not") {
		t.Fatal("expected no tag match")
	}
}

func TestSetTags(t *testing.T) {
	w := NewReaderWay(1)
	m := map[string]any{"test": "xy"}
	w.SetTags(m)
	if !w.HasTag("test", "xy") {
		t.Fatal("expected tag after SetTags")
	}
	w.SetTags(nil)
	if w.HasTag("test", "xy") {
		t.Fatal("expected no tags after SetTags(nil)")
	}
}

func TestInvalidIDs(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for negative ID")
		}
		msg, ok := r.(string)
		if !ok || len(msg) == 0 {
			t.Fatal("expected non-empty panic message")
		}
		expected := "Invalid OSM WAY Id: -1;"
		if len(msg) < len(expected) || msg[:len(expected)] != expected {
			t.Fatalf("unexpected panic message: %s", msg)
		}
	}()
	NewReaderWay(-1)
}

func TestGetFirstValue(t *testing.T) {
	w := NewReaderWay(1)
	w.SetTag("name:en", "Berlin")
	w.SetTag("name", "Berlin (de)")
	v := w.GetFirstValue([]string{"name:en", "name"})
	if v != "Berlin" {
		t.Fatalf("expected 'Berlin', got %q", v)
	}
}

func TestGetFirstIndex(t *testing.T) {
	w := NewReaderWay(1)
	w.SetTag("ref", "A1")
	idx := w.GetFirstIndex([]string{"name", "ref"})
	if idx != 1 {
		t.Fatalf("expected index 1, got %d", idx)
	}
	idx = w.GetFirstIndex([]string{"missing"})
	if idx != -1 {
		t.Fatalf("expected -1, got %d", idx)
	}
}

func TestReaderRelation(t *testing.T) {
	r := NewReaderRelation(42)
	if len(r.GetMembers()) != 0 {
		t.Fatal("expected empty members")
	}
	r.Add(Member{Type: TypeWay, Ref: 100, Role: "from"})
	r.Add(Member{Type: TypeNode, Ref: 200, Role: "via"})
	if len(r.GetMembers()) != 2 {
		t.Fatalf("expected 2 members, got %d", len(r.GetMembers()))
	}
	if r.IsMetaRelation() {
		t.Fatal("expected non-meta relation")
	}
	r.Add(Member{Type: TypeRelation, Ref: 300, Role: ""})
	if !r.IsMetaRelation() {
		t.Fatal("expected meta relation")
	}
}

func TestReaderNode(t *testing.T) {
	n := NewReaderNode(1, 52.5, 13.4)
	if n.Lat != 52.5 || n.Lon != 13.4 {
		t.Fatalf("unexpected coords: %f, %f", n.Lat, n.Lon)
	}
	if n.GetID() != 1 {
		t.Fatalf("expected id 1, got %d", n.GetID())
	}
	if n.GetType() != TypeNode {
		t.Fatalf("expected NODE type, got %s", n.GetType())
	}
}
