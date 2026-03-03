package osm

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"gohopper/core/reader"

	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmpbf"
	"github.com/paulmach/osm/osmxml"
)

// ReadFile reads an OSM file (PBF or XML) and dispatches elements to the handler.
// Elements are yielded in file order. SkipOptions controls which types to skip.
func ReadFile(path string, handler ElementHandler, skip SkipOptions) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot open OSM file %q: %w", path, err)
	}
	defer f.Close()

	scanner, err := newScanner(f, path, skip)
	if err != nil {
		return err
	}
	defer scanner.Close()

	for scanner.Scan() {
		obj := scanner.Object()
		switch o := obj.(type) {
		case *osm.Node:
			if skip.SkipNodes {
				continue
			}
			handler.HandleNode(convertNode(o))
		case *osm.Way:
			if skip.SkipWays {
				continue
			}
			handler.HandleWay(convertWay(o))
		case *osm.Relation:
			if skip.SkipRelations {
				continue
			}
			handler.HandleRelation(convertRelation(o))
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading OSM file %q: %w", path, err)
	}

	handler.OnFinish()
	return nil
}

// osmScanner abstracts over osmpbf.Scanner and osmxml.Scanner.
type osmScanner interface {
	Scan() bool
	Object() osm.Object
	Err() error
	Close() error
}

func newScanner(r io.ReadSeeker, path string, skip SkipOptions) (osmScanner, error) {
	if isPBF(path) {
		s := osmpbf.New(context.Background(), r, runtime.GOMAXPROCS(0))
		s.SkipNodes = skip.SkipNodes
		s.SkipWays = skip.SkipWays
		s.SkipRelations = skip.SkipRelations
		return s, nil
	}
	// Fall back to XML for .osm / .xml files.
	// Reset read position in case we peeked.
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return osmxml.New(context.Background(), r), nil
}

func isPBF(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".pbf") || strings.HasSuffix(lower, ".osm.pbf")
}

func convertNode(n *osm.Node) *reader.ReaderNode {
	node := reader.NewReaderNode(int64(n.ID), n.Lat, n.Lon)
	for _, t := range n.Tags {
		node.SetTag(t.Key, t.Value)
	}
	return node
}

func convertWay(w *osm.Way) *reader.ReaderWay {
	way := reader.NewReaderWay(int64(w.ID))
	way.Nodes = make([]int64, len(w.Nodes))
	for i, n := range w.Nodes {
		way.Nodes[i] = int64(n.ID)
	}
	for _, t := range w.Tags {
		way.SetTag(t.Key, t.Value)
	}
	return way
}

func convertRelation(r *osm.Relation) *reader.ReaderRelation {
	rel := reader.NewReaderRelation(int64(r.ID))
	for _, m := range r.Members {
		var mType reader.ElementType
		switch m.Type {
		case osm.TypeNode:
			mType = reader.TypeNode
		case osm.TypeWay:
			mType = reader.TypeWay
		case osm.TypeRelation:
			mType = reader.TypeRelation
		}
		rel.Add(reader.Member{
			Type: mType,
			Ref:  int64(m.Ref),
			Role: string(m.Role),
		})
	}
	for _, t := range r.Tags {
		rel.SetTag(t.Key, t.Value)
	}
	return rel
}
