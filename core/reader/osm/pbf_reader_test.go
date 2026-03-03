package osm

import (
	"testing"

	"gohopper/core/reader"
)

type countingHandler struct {
	nodes     int
	ways      int
	relations int
	headers   int
	finished  bool

	firstNode *reader.ReaderNode
	firstWay  *reader.ReaderWay
}

func (h *countingHandler) HandleNode(n *reader.ReaderNode) {
	if h.nodes == 0 {
		h.firstNode = n
	}
	h.nodes++
}

func (h *countingHandler) HandleWay(w *reader.ReaderWay) {
	if h.ways == 0 {
		h.firstWay = w
	}
	h.ways++
}

func (h *countingHandler) HandleRelation(_ *reader.ReaderRelation) { h.relations++ }
func (h *countingHandler) HandleFileHeader(_ *OSMFileHeader)       { h.headers++ }
func (h *countingHandler) OnFinish()                               { h.finished = true }

func TestReadPBF(t *testing.T) {
	h := &countingHandler{}
	err := ReadFile("testdata/test-osm6.pbf", h, SkipOptionsNone())
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !h.finished {
		t.Fatal("OnFinish not called")
	}
	if h.nodes == 0 {
		t.Fatal("expected nodes > 0")
	}
	if h.ways == 0 {
		t.Fatal("expected ways > 0")
	}
	t.Logf("PBF: nodes=%d, ways=%d, relations=%d", h.nodes, h.ways, h.relations)
}

func TestReadPBFSkipNodes(t *testing.T) {
	h := &countingHandler{}
	err := ReadFile("testdata/test-osm6.pbf", h, SkipOptions{SkipNodes: true})
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if h.nodes != 0 {
		t.Fatalf("expected 0 nodes when skipping, got %d", h.nodes)
	}
	if h.ways == 0 {
		t.Fatal("expected ways > 0 even when skipping nodes")
	}
}

func TestReadXML(t *testing.T) {
	h := &countingHandler{}
	err := ReadFile("testdata/test-osm.xml", h, SkipOptionsNone())
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !h.finished {
		t.Fatal("OnFinish not called")
	}
	if h.nodes == 0 {
		t.Fatal("expected nodes > 0")
	}
	if h.ways == 0 {
		t.Fatal("expected ways > 0")
	}
	t.Logf("XML: nodes=%d, ways=%d, relations=%d", h.nodes, h.ways, h.relations)
}

func TestReadXMLTags(t *testing.T) {
	h := &countingHandler{}
	err := ReadFile("testdata/test-osm.xml", h, SkipOptionsNone())
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	// The first way in test-osm.xml should have a highway tag.
	if h.firstWay == nil {
		t.Fatal("expected at least one way")
	}
	if len(h.firstWay.GetNodes()) == 0 {
		t.Fatal("expected way to have nodes")
	}
}
