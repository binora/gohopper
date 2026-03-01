package routing

import (
	"testing"

	"gohopper/core/util"
)

func TestSPTEntry_NewDefaults(t *testing.T) {
	e := NewSPTEntry(5, 1.5)
	if e.Edge != util.NoEdge {
		t.Fatalf("expected Edge=%d, got %d", util.NoEdge, e.Edge)
	}
	if e.AdjNode != 5 {
		t.Fatalf("expected AdjNode=5, got %d", e.AdjNode)
	}
	if e.Weight != 1.5 {
		t.Fatalf("expected Weight=1.5, got %f", e.Weight)
	}
	if e.Parent != nil {
		t.Fatal("expected Parent to be nil")
	}
	if e.Deleted {
		t.Fatal("expected Deleted to be false")
	}
}

func TestSPTEntry_Less(t *testing.T) {
	a := NewSPTEntry(0, 1.0)
	b := NewSPTEntry(1, 2.0)
	if !a.Less(b) {
		t.Fatal("expected a < b")
	}
	if b.Less(a) {
		t.Fatal("expected b >= a")
	}

	c := NewSPTEntry(2, 1.0)
	if a.Less(c) || c.Less(a) {
		t.Fatal("expected equal weights to not be less")
	}
}

func TestSPTEntry_ParentChain(t *testing.T) {
	root := NewSPTEntry(0, 0.0)
	mid := NewSPTEntryFull(10, 1, 1.0, root)
	leaf := NewSPTEntryFull(20, 2, 2.0, mid)

	if leaf.Parent != mid {
		t.Fatal("expected leaf.Parent == mid")
	}
	if mid.Parent != root {
		t.Fatal("expected mid.Parent == root")
	}
	if root.Parent != nil {
		t.Fatal("expected root.Parent == nil")
	}

	// Walk the chain and verify adj nodes
	expected := []int{2, 1, 0}
	cur := leaf
	for i, want := range expected {
		if cur == nil {
			t.Fatalf("chain ended early at index %d", i)
		}
		if cur.AdjNode != want {
			t.Fatalf("at index %d: expected AdjNode=%d, got %d", i, want, cur.AdjNode)
		}
		cur = cur.Parent
	}
	if cur != nil {
		t.Fatal("expected chain to end after root")
	}
}

func TestSPTEntry_Deleted(t *testing.T) {
	e := NewSPTEntry(0, 0.0)
	if e.Deleted {
		t.Fatal("expected Deleted to be false initially")
	}
	e.Deleted = true
	if !e.Deleted {
		t.Fatal("expected Deleted to be true after setting")
	}
}
